package engine

import (
	"context"
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

func (r *HTTPRuntime) HandleTLSConnection(ctx context.Context, e string, conn *tls.Conn) error {
	handler, ok := r.handlers[e]
	if !ok {
		conn.Close()
		return fmt.Errorf("No handlers registered for this entrypoint")
	}
	srv := &http.Server{
		Handler:     handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	err := srv.Serve(&singleConnListener{conn: conn})
	if err != io.EOF {
		return err
	}
	return nil
}

func (r *HTTPRuntime) HandleRawConnection(ctx context.Context, e string, conn BufferedConn) error {
	handler, ok := r.handlers[e]
	if !ok {
		return fmt.Errorf("No handlers registered for this entrypoint")
	}
	srv := &http.Server{
		Handler:     handler,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	err := srv.Serve(&singleConnListener{conn: conn})
	if err != io.EOF {
		return err
	}
	return nil
}

func (r *HTTPRuntime) Claim(e string, conn BufferedConn) bool {
	if _, ok := r.handlers[e]; !ok {
		return false
	}

	if ok, _ := CheckForClientHello(conn); ok {
		return true
	}

	data, err := conn.Peek(5)
	if err != nil {
		return false
	}

	s := string(data)

	switch s {
	case
		"GET /",
		"HEAD ",
		"POST ",
		"PUT /",
		"DELET",
		"CONNE",
		"OPTIO",
		"TRACE",
		"PATCH":
		return true
	}

	return false
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
