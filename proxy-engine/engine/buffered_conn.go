package engine

import (
	"bufio"
	"net"
	"time"
)

type BufferedConn interface {
	Read(p []byte) (int, error)
	Peek(n int) ([]byte, error)
	Write(b []byte) (n int, err error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	NetConn() net.Conn
	Reader() *bufio.Reader
}

type bufferedConn struct {
	conn   net.Conn
	reader *bufio.Reader
}

func NewBufferedConn(conn net.Conn) BufferedConn {
	r := bufio.NewReader(conn)

	if r == nil {
		return nil
	}

	return &bufferedConn{
		conn:   conn,
		reader: r,
	}
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *bufferedConn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func (c *bufferedConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

func (c *bufferedConn) Close() error {
	return c.conn.Close()
}

func (c *bufferedConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *bufferedConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *bufferedConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *bufferedConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *bufferedConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *bufferedConn) NetConn() net.Conn {
	return c.conn
}

func (c *bufferedConn) Reader() *bufio.Reader {
	return c.reader
}
