package proxy

import "net"

type TCPEntryPoint struct {
	Identifer string
	Address   string
}

func (e *TCPEntryPoint) Listen() (net.Listener, error) {
	return net.Listen("tcp", e.Address)
}

func (e *TCPEntryPoint) Id() string {
	return e.Identifer
}
