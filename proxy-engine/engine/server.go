package engine

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"sync"
)

type Server struct {
	mu                sync.Mutex
	listeners         map[string]net.Listener
	tlsConfigHandlers map[string]TLSConfigHandler
	addEntryPoint     chan EntryPoint
	removeEntryPoint  chan string
	filter            ConnFilter

	tcpRuntime  *TCPRuntime
	httpRuntime *HTTPRuntime
}

const initBufferSize = 100

func NewServer() *Server {
	return &Server{
		listeners:         make(map[string]net.Listener),
		tlsConfigHandlers: make(map[string]TLSConfigHandler),
		addEntryPoint:     make(chan EntryPoint, initBufferSize),
		removeEntryPoint:  make(chan string, initBufferSize),
		tcpRuntime:        NewTCPRuntime(),
		httpRuntime:       NewHTTPRuntime(),
		filter:            nil,
	}
}

func (s *Server) SetFilter(filter ConnFilter) {
	s.filter = filter
}

func (s *Server) Serve(ctx context.Context) error {
	for {
		select {
		case e := <-s.addEntryPoint:
			s.startEntryPoint(ctx, e)
		case id := <-s.removeEntryPoint:
			s.stopEntryPoint(id)
		}
	}
}

func (s *Server) acceptLoop(ctx context.Context, e string, ln net.Listener) {
	cancelCtx, cancel := context.WithCancel(ctx)

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				cancel()
				return
			default:
			}
		}

		if conn == nil {
			cancel()
			return
		}

		go s.handleConnection(cancelCtx, e, conn)
	}
}

func (s *Server) handleTLSConnection(ctx context.Context, e string, conn BufferedConn) {
	log.Printf("%s | Handling connection as TLS\n", conn.RemoteAddr().String())

	transport, err := GetTransport(conn.LocalAddr())
	if err != nil {
		return
	}

	switch transport {
	case TransportTCP:
		if s.tcpRuntime.Claim(e, NewTCPContext(conn)) {
			s.tcpRuntime.Handle(ctx, e, conn)
			return
		}
	default:
		log.Printf(
			"%s | TLS connection occuring over unsupported transport protocol: %s\n",
			conn.RemoteAddr().String(),
			transport,
		)
		conn.Close()
		return
	}

	// terminate tls and check if http router claims

	tlsConfigHandler, ok := s.tlsConfigHandlers[e]
	if !ok {
		log.Printf(
			"%s | TLS config compiler not configured for the entrypoint %s\n",
			conn.RemoteAddr().String(),
			e,
		)
		conn.Close()
		return
	}

	tlsInfo, err := PeekTLSClientHelloInfo(conn)
	if err != nil {
		conn.Close()
		return
	}

	tlsConfig, err := tlsConfigHandler.HandleTLSConfig(tlsInfo)
	if err != nil {
		log.Printf(
			"%s | TLS config compiler failed to serve TLS config with error: %s\n",
			conn.RemoteAddr().String(),
			err,
		)
		conn.Close()
		return
	}

	tlsConn := tls.Server(conn, tlsConfig)

	err = tlsConn.Handshake()
	if err != nil {
		log.Printf(
			"%s | TLS handshake failed with error: %s\n",
			conn.RemoteAddr().String(),
			err,
		)
		conn.Close()
		return
	}

	// Assume protocol is https
	err = s.httpRuntime.HandleTLSConnection(ctx, e, tlsConn)
	if err != nil {
		log.Printf(
			"%s | HTTP runtime failed to handle TLS connection with error: %s\n",
			conn.RemoteAddr().String(),
			err,
		)
		conn.Close()
	}
}

func (s *Server) handleRawConnection(ctx context.Context, e string, conn BufferedConn) {
	log.Printf("%s | Handling connection as raw\n", conn.RemoteAddr().String())

	transport, err := GetTransport(conn.LocalAddr())
	if err != nil {
		conn.Close()
		return
	}

	// Check if connection is HTTP/1.1 and a match
	if s.httpRuntime.Claim(e, conn) {
		log.Printf(
			"%s | Raw connection determined to be HTTP/1.1\n",
			conn.RemoteAddr().String(),
		)
		err = s.httpRuntime.HandleRawConnection(ctx, e, conn)
		if err != nil {
			log.Printf(
				"%s | HTTP runtime failed to handle raw connection with error: %s\n",
				conn.RemoteAddr().String(),
				err,
			)
			conn.Close()
		}
		return
	}

	// Otherwise send to fallback
	switch transport {
	case TransportTCP:
		if s.tcpRuntime.Claim(e, NewTCPContext(conn)) {
			s.tcpRuntime.Handle(ctx, e, conn)
			return
		}
	case TransportUDP:
		// not implemented
	case TransportUnix:
		// not implemented
	}

	log.Printf("%s | Could not determine a runtime to handle request\n", conn.RemoteAddr())
	conn.Close()
}

type ConnFilter interface {
	KeepConnection(net.Conn) bool
}

type ConnFilterFunc func(conn net.Conn) bool

func (f ConnFilterFunc) KeepConnection(conn net.Conn) bool {
	return f(conn)
}

func (s *Server) handleConnection(ctx context.Context, e string, conn net.Conn) {
	if conn == nil {
		log.Println("net.Conn is nil")
		return
	}

	if s.filter != nil && !s.filter.KeepConnection(conn) {
		conn.Close()
		return
	}

	bufferedConn := NewBufferedConn(conn)
	if bufferedConn == nil {
		conn.Close()
		return
	}

	log.Printf(
		"%s | Connection recieved to entrypoint \"%s\"\n",
		conn.RemoteAddr().String(),
		e,
	)

	isClientHello, err := CheckForClientHello(bufferedConn)
	if err != nil {
		conn.Close()
		return
	}

	// Connection using tls, need to figure out if decryption is needed.
	// It is the Servers sole responsibility to handle decryption when needed.
	// The Server can ask the routers what certs to use.
	if isClientHello {
		if _, ok := s.tlsConfigHandlers[e]; ok {
			s.handleTLSConnection(ctx, e, bufferedConn)
		} else {
			log.Printf(
				"%s | TLS connection recieved but no config compiler is available to handle it\n",
				conn.RemoteAddr().String(),
			)
		}
	} else {
		s.handleRawConnection(ctx, e, bufferedConn)
	}
}

func (s *Server) startEntryPoint(ctx context.Context, e EntryPoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.listeners[e.Id()]; exists {
		return
	}

	ln, err := e.Listen()
	if err != nil {
		log.Printf("Failed to listen with error: %s\n", err)
		return
	}

	s.listeners[e.Id()] = ln

	go s.acceptLoop(ctx, e.Id(), ln)
}

func (s *Server) stopEntryPoint(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ln, ok := s.listeners[id]; ok {
		ln.Close()
		delete(s.listeners, id)
	}
}

func (s *Server) Shutdown(ctx context.Context) {

}

func (s *Server) RegisterEntryPoint(e EntryPoint) {
	s.addEntryPoint <- e
}

func (s *Server) DeregisterEntryPoint(id string) {
	s.removeEntryPoint <- id
}

func (s *Server) RegisterHTTPHandler(entryPointId string, handler http.Handler) {
	s.httpRuntime.RegisterHandler(entryPointId, handler)
}

func (s *Server) DeregisterHTTPHandler(entryPointId string) {
	s.httpRuntime.DeregisterHandler(entryPointId)
}

func (s *Server) RegisterTCPHandler(entryPointId string, handler TCPHandler) {
	s.tcpRuntime.RegisterHandler(entryPointId, handler)
}

func (s *Server) DeregisterTCPHandler(entryPointId string) {
	s.tcpRuntime.DeregisterHandler(entryPointId)
}

func (s *Server) RegisterTLSConfigHandler(entryPointId string, tls TLSConfigHandler) {
	s.tlsConfigHandlers[entryPointId] = tls
}

func (s *Server) DeregisterTLSConfigHandler(entryPointId string) {
	delete(s.tlsConfigHandlers, entryPointId)
}
