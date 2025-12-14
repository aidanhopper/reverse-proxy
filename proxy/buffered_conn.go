package proxy

import (
	"bufio"
	"net"
	"time"
)

type BufferedConn struct {
	conn   net.Conn
	reader *bufio.Reader
	// writer *bufio.Writer
}

func NewBufferedConn(conn net.Conn) *BufferedConn {
	return &BufferedConn{
		conn:   conn,
		reader: bufio.NewReader(conn),
		// writer: bufio.NewWriter(conn),
	}
}

func (c *BufferedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *BufferedConn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func (c *BufferedConn) Write(b []byte) (n int, err error) {
	return c.conn.Write(b)
}

// func (c *BufferedConn) Flush() error {
// 	return c.writer.Flush()
// }

func (c *BufferedConn) Close() error {
	return c.conn.Close()
}

func (c *BufferedConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *BufferedConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *BufferedConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *BufferedConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *BufferedConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
