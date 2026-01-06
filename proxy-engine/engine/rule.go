package engine

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

type Rule interface {
	Match(any) bool
}

type RuleFunc func(any) bool

func (f RuleFunc) Match(v any) bool {
	return f(v)
}

type HTTPRuleFunc func(*http.Request) bool

type TCPRuleFunc func(*TCPContext) bool

func (r HTTPRuleFunc) Match(v any) bool {
	req, ok := v.(*http.Request)
	if !ok {
		return false
	}
	return r(req)
}

func (r TCPRuleFunc) Match(v any) bool {
	ctx, ok := v.(*TCPContext)
	if !ok {
		return false
	}
	return r(ctx)
}

func Host(host string) HTTPRuleFunc {
	return func(r *http.Request) bool {
		return r.Host == host
	}
}

func PathPrefix(prefix string) HTTPRuleFunc {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

func PathRegexp(pattern string) HTTPRuleFunc {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return func(r *http.Request) bool {
			return false
		}
	}

	return func(r *http.Request) bool {
		return re.MatchString(r.URL.Path)
	}
}

func Path(path string) HTTPRuleFunc {
	return func(r *http.Request) bool {
		return r.URL.Path == path
	}
}

func And(rules ...Rule) RuleFunc {
	return RuleFunc(func(v any) bool {
		for _, r := range rules {
			if !r.Match(v) {
				return false
			}
		}
		return true
	})
}

func Or(rules ...Rule) RuleFunc {
	return RuleFunc(func(v any) bool {
		for _, r := range rules {
			if r.Match(v) {
				return true
			}
		}
		return false
	})
}

func Not(r Rule) RuleFunc {
	return RuleFunc(func(v any) bool {
		return !r.Match(v)
	})
}

func Any() Rule {
	return RuleFunc(func(_ any) bool {
		return true
	})
}

func Method(method string) HTTPRuleFunc {
	return func(r *http.Request) bool {
		return r.Method == method
	}
}

func HostSNI(sni string) TCPRuleFunc {
	return func(t *TCPContext) bool {
		return t.ProtoType == "TLS" && t.SNI == sni
	}
}

func decodeVarInt(data []byte) (value int, length int, err error) {
	if len(data) == 0 {
		return 0, 0, io.EOF
	}

	for i := range data {
		b := data[i]
		value |= int(b&0x7F) << (7 * i)
		length = i + 1
		if (b & 0x80) == 0 {
			return value, length, nil
		}
		if length >= 5 {
			return 0, 0, errors.New("VarInt too large")
		}
	}
	return 0, 0, errors.New("incomplete VarInt")
}

type minecraftHandshakeData struct {
	RequestedHost   string
	RequestedPort   uint16
	ProtocolState   int
	ProtocolVersion int

	Username     string
	IsLoginStart bool
}

func extractMinecraftData(t *TCPContext) (minecraftHandshakeData, error) {
	result := minecraftHandshakeData{}

	const maxVarIntSize = 5
	const defensiveMaxPacketSize = 8192

	data, err := (*t).Peek(maxVarIntSize)
	if err != nil && err != io.EOF {
		return result, err
	}
	if len(data) < maxVarIntSize {
		return result, nil
	}

	handshakePayloadLen, prefixLen, err := decodeVarInt(data)
	if err != nil {
		return result, errors.New("malformed handshake length prefix")
	}

	totalHandshakeLen := prefixLen + handshakePayloadLen

	if totalHandshakeLen > defensiveMaxPacketSize {
		return result, errors.New("handshake size exceeds safety limit")
	}

	data, err = (*t).Peek(totalHandshakeLen)
	if err != nil && err != io.EOF {
		return result, err
	}
	if len(data) < totalHandshakeLen {
		return result, nil
	}

	offset := prefixLen
	// Skip Packet ID (0x00)
	_, idLen, err := decodeVarInt(data[offset:])
	if err != nil {
		return result, errors.New("malformed handshake ID")
	}
	offset += idLen
	// Decode Protocol Version
	version, versionLen, err := decodeVarInt(data[offset:])
	if err != nil {
		return result, errors.New("malformed protocol version")
	}
	result.ProtocolVersion = version
	offset += versionLen
	// Decode Server Address (Host String)
	hostLen, varIntLen, err := decodeVarInt(data[offset:])
	if err != nil {
		return result, errors.New("malformed host length")
	}
	offset += varIntLen
	// Extract Requested Host
	result.RequestedHost = string(data[offset : offset+hostLen])
	offset += hostLen
	// Decode Server Port (2 bytes)
	result.RequestedPort = uint16(data[offset])<<8 | uint16(data[offset+1])
	offset += 2
	// Decode Next State (1=Status, 2=Login)
	nextState, stateLen, err := decodeVarInt(data[offset:])
	if err != nil {
		return result, errors.New("malformed next state")
	}
	result.ProtocolState = nextState
	offset += stateLen

	if result.ProtocolState != 2 {
		return result, nil // Not a Login attempt, stop here.
	}

	// Peek enough bytes past the handshake end to guarantee we can decode the
	// Packet Length, Packet ID, and Username Length (3 VarInts, max 15 bytes)
	const loginPrefixCheck = 15

	extendedData, err := (*t).Peek(offset + loginPrefixCheck)
	if err != nil && err != io.EOF {
		return result, err
	}

	loginData := extendedData[offset:]
	if len(loginData) < loginPrefixCheck {
		return result, nil
	}

	loginOffset := 0

	_, lLoginLen, err := decodeVarInt(loginData[loginOffset:])
	if err != nil {
		return result, nil
	}
	loginOffset += lLoginLen

	packetID, packetIDLen, err := decodeVarInt(loginData[loginOffset:])
	if err != nil {
		return result, nil
	}
	loginOffset += packetIDLen

	if packetID != 0 {
		return result, nil
	}

	result.IsLoginStart = true

	userLen, userLenLen, err := decodeVarInt(loginData[loginOffset:])
	if err != nil {
		return result, errors.New("malformed username length")
	}
	loginOffset += userLenLen

	totalUsernameBytesNeeded := offset + loginOffset + userLen

	finalData, err := (*t).Peek(totalUsernameBytesNeeded)
	if err != nil && err != io.EOF {
		return result, err
	}

	if len(finalData) < totalUsernameBytesNeeded {
		return result, nil
	}

	// Extract Username String
	usernameStart := offset + loginOffset

	result.Username = string(finalData[usernameStart : usernameStart+userLen])

	return result, nil
}

func HostMinecraft(hosts ...string) TCPRuleFunc {
	return func(t *TCPContext) bool {
		data, err := extractMinecraftData(t)
		if err != nil {
			return false
		}

		return slices.Contains(hosts, data.RequestedHost)
	}
}

func PlayerMinecraft(players ...string) TCPRuleFunc {
	return func(t *TCPContext) bool {
		data, err := extractMinecraftData(t)
		if err != nil {
			return false
		}

		if !data.IsLoginStart {
			return true
		}

		return slices.Contains(players, data.Username)
	}
}


func NotPlayerMinecraft(players ...string) TCPRuleFunc {
	return func(t *TCPContext) bool {
		data, err := extractMinecraftData(t)
		if err != nil {
			return false
		}

		if !data.IsLoginStart {
			return true
		}

		return !slices.Contains(players, data.Username)
	}
}
