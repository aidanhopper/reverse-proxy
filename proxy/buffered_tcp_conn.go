package proxy

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type BufferedTCPConn struct {
	tcpConn *net.TCPConn
	reader  *bufio.Reader
}

func NewBufferedTCPConn(bconn BufferedConn) (*BufferedTCPConn, error) {
	tcpConn, ok := bconn.NetConn().(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("underlying connection is not a *net.TCPConn")
	}

	return &BufferedTCPConn{
		tcpConn: tcpConn,
		reader:  bconn.Reader(),
	}, nil
}

func (b *BufferedTCPConn) Read(p []byte) (n int, err error) {
	return b.reader.Read(p)
}

func (b *BufferedTCPConn) Write(p []byte) (n int, err error) {
	return b.tcpConn.Write(p)
}

func (b *BufferedTCPConn) NetConn() net.Conn {
	return b.tcpConn
}

func (c *BufferedTCPConn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func (c *BufferedTCPConn) Reader() *bufio.Reader {
	return c.reader
}

func (c *BufferedTCPConn) Close() error {
	return c.tcpConn.Close()
}

func (c *BufferedTCPConn) LocalAddr() net.Addr {
	return c.tcpConn.LocalAddr()
}

func (c *BufferedTCPConn) RemoteAddr() net.Addr {
	return c.tcpConn.RemoteAddr()
}

func (c *BufferedTCPConn) SetDeadline(t time.Time) error {
	return c.tcpConn.SetDeadline(t)
}

func (c *BufferedTCPConn) SetReadDeadline(t time.Time) error {
	return c.tcpConn.SetReadDeadline(t)
}

func (c *BufferedTCPConn) SetWriteDeadline(t time.Time) error {
	return c.tcpConn.SetWriteDeadline(t)
}
