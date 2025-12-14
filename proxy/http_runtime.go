package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
)

type HTTPRuntime struct {
	handlers map[string]http.Handler
}

func NewHTTPRuntime() *HTTPRuntime {
	return &HTTPRuntime{
		handlers: make(map[string]http.Handler),
	}
}

type singleConnListener struct {
	conn net.Conn
	done bool
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.done {
		return nil, io.EOF
	}
	l.done = true
	return l.conn, nil
}

func (l *singleConnListener) Close() error   { return nil }
func (l *singleConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

func (r *HTTPRuntime) HandleTLSConnection(e EntryPoint, conn *tls.Conn) error {
	handler, ok := r.handlers[e.Id()]
	if !ok {
		conn.Close()
		return fmt.Errorf("No handlers registered for this entrypoint")
	}
	srv := &http.Server{Handler: handler}
	err := srv.Serve(&singleConnListener{conn: conn})
	if err != io.EOF {
		return err
	}
	return nil
}

func (r *HTTPRuntime) HandleRawConnection(e EntryPoint, conn *BufferedConn) error {
	handler, ok := r.handlers[e.Id()]
	if !ok {
		conn.Close()
		return fmt.Errorf("No handlers registered for this entrypoint")
	}
	srv := &http.Server{Handler: handler}
	err := srv.Serve(&singleConnListener{conn: conn})
	if err != io.EOF {
		return err
	}
	return nil
}

func (r *HTTPRuntime) Claim(conn *BufferedConn) bool {
	// will claim a HTTP/1.1 connection when needed
	return true
}

func (r *HTTPRuntime) RegisterHandler(entryPointId string, handler http.Handler) {
	r.handlers[entryPointId] = handler
}

func (r *HTTPRuntime) DeregisterHandler(entryPointId string) {
	delete(r.handlers, entryPointId)
}

func (r *HTTPRuntime) IsHandlerRegistered(entryPointId string) bool {
	_, present := r.handlers[entryPointId]
	return present
}
