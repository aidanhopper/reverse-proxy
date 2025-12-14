package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
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

func PeekTLSConnectionState(conn *BufferedConn) (tls.ConnectionState, error) {
	const tlsRecordHeaderLen = 5
	peekedHeader, err := conn.Peek(tlsRecordHeaderLen)

	if peekedHeader[0] != 0x16 {
		return tls.ConnectionState{}, fmt.Errorf("not a TLS handshake record (%x)", peekedHeader[0])
	}

	payloadLen := int(peekedHeader[3])<<8 | int(peekedHeader[4])
	totalPeekLen := tlsRecordHeaderLen + payloadLen

	peekedBytes, err := conn.Peek(totalPeekLen)
	if err != nil {
		if len(peekedBytes) == 0 || err != io.EOF {
			return tls.ConnectionState{}, err
		}
	}

	c := &peekConn{
		Reader: bytes.NewReader(peekedBytes),
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{{}},
	}

	tlsConn := tls.Server(c, config)

	err = tlsConn.Handshake()
	if err != nil {
		return tls.ConnectionState{}, err
	}

	return tlsConn.ConnectionState(), nil
}
