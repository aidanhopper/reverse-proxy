package proxy

import (
	"io"
)

type TCPRuntime struct {
}

func (r *TCPRuntime) Claim(conn *BufferedConn) bool {
	// sni, _ := PeekSNI(conn)
	return false
}

func (r *TCPRuntime) Handle(conn *BufferedConn) {
	defer conn.Close()
	io.Copy(conn, conn)
}
