package proxy

import (
	"context"
	"fmt"
)

type TCPRuntime struct {
	handlers map[string]TCPHandler
}

func NewTCPRuntime() *TCPRuntime {
	return &TCPRuntime{
		make(map[string]TCPHandler),
	}
}

func (r *TCPRuntime) Claim(e string, ctx *TCPContext) bool {
	handler, present := r.handlers[e]
	if !present {
		return false
	}
	return handler.Rule().Match(ctx)
}

func (r *TCPRuntime) Handle(ctx context.Context, e string, bconn BufferedConn) error {
	conn, err := NewBufferedTCPConn(bconn)
	if err != nil {
		conn.Close()
		return fmt.Errorf("Failed to create TCP connection object. Is the transport protcol not TCP?")
	}

	handler, ok := r.handlers[e]
	if !ok {
		conn.Close()
		return fmt.Errorf("No handlers registered for this entrypoint")
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		handler.ServeTCP(conn)
	}()

	select {
	case <-ctx.Done():
		conn.Close()
		return fmt.Errorf("Context is done")
	case <-done:
		return nil
	}
}

func (r *TCPRuntime) RegisterHandler(entryPointId string, handler TCPHandler) {
	r.handlers[entryPointId] = handler
}

func (r *TCPRuntime) DeregisterHandler(entryPointId string) {
	delete(r.handlers, entryPointId)
}

func (r *TCPRuntime) IsHandlerRegistered(entryPointId string) bool {
	_, present := r.handlers[entryPointId]
	return present
}
