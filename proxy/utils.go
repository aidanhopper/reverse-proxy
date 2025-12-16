package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type peekConn struct {
	io.Reader
	net.Conn
}

func (pc *peekConn) Read(b []byte) (n int, err error) {
	return pc.Reader.Read(b)
}

func (pc *peekConn) Write(b []byte) (int, error)        { return 0, io.EOF }
func (pc *peekConn) Close() error                       { return nil }
func (pc *peekConn) LocalAddr() net.Addr                { return nil }
func (pc *peekConn) RemoteAddr() net.Addr               { return nil }
func (pc *peekConn) SetDeadline(t time.Time) error      { return nil }
func (pc *peekConn) SetReadDeadline(t time.Time) error  { return nil }
func (pc *peekConn) SetWriteDeadline(t time.Time) error { return nil }

func PeekTLSClientHelloInfo(conn BufferedConn) (*tls.ClientHelloInfo, error) {
	const tlsRecordHeaderLen = 5
	peekedHeader, err := conn.Peek(tlsRecordHeaderLen)

	if peekedHeader[0] != 0x16 {
		return nil, fmt.Errorf("not a TLS handshake record (%x)", peekedHeader[0])
	}

	payloadLen := int(peekedHeader[3])<<8 | int(peekedHeader[4])
	totalPeekLen := tlsRecordHeaderLen + payloadLen

	peekedBytes, err := conn.Peek(totalPeekLen)
	if err != nil {
		if len(peekedBytes) == 0 || err != io.EOF {
			return nil, err
		}
	}

	c := &peekConn{
		Reader: bytes.NewReader(peekedBytes),
	}

	var state tls.ClientHelloInfo
	config := &tls.Config{
		Certificates: []tls.Certificate{{}},
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			state = *info
			return nil, io.EOF
		},
	}

	tlsConn := tls.Server(c, config)

	err = tlsConn.Handshake()
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &state, nil
}

func CheckForClientHello(conn BufferedConn) (bool, error) {
	bytes, err := conn.Peek(1)
	if err != nil {
		return false, err
	}
	return bytes[0] == 0x16, nil
}

type TCPContext struct {
	SNI         string
	LocalAddr   net.Addr
	ClientAddr  net.Addr
	RemoteAddr  net.Addr
	RemoteIP    string
	ClaimedPort string
	ProtoType   string
	Peek        func(n int) ([]byte, error)
}

func NewTCPContext(conn BufferedConn) *TCPContext {
	ctx := TCPContext{
		LocalAddr:  conn.LocalAddr(),
		ClientAddr: conn.RemoteAddr(),
		RemoteAddr: conn.RemoteAddr(),
		ProtoType:  "TCP",
		Peek:       conn.Reader().Peek,
	}

	if tcpAddr, ok := ctx.ClientAddr.(*net.TCPAddr); ok {
		ctx.RemoteIP = tcpAddr.IP.String()
	} else if addr := ctx.ClientAddr.String(); strings.Contains(addr, ":") {
		if host, _, err := net.SplitHostPort(addr); err == nil {
			ctx.RemoteIP = host
		} else {
			ctx.RemoteIP = addr
		}
	} else {
		ctx.RemoteIP = ctx.ClientAddr.String()
	}

	// Parse Claimed Port
	if tcpAddr, ok := ctx.LocalAddr.(*net.TCPAddr); ok {
		ctx.ClaimedPort = fmt.Sprintf("%d", tcpAddr.Port)
	} else {
		if _, port, err := net.SplitHostPort(ctx.LocalAddr.String()); err == nil {
			ctx.ClaimedPort = port
		} else {
			ctx.ClaimedPort = ctx.LocalAddr.String()
		}
	}

	state, err := PeekTLSClientHelloInfo(conn)

	if err == nil {
		ctx.ProtoType = "TLS"
		ctx.SNI = state.ServerName
	}

	return &ctx
}
