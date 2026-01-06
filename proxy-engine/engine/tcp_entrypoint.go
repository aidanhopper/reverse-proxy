package engine

import (
	"net"
)

type TCPEntryPoint struct {
	Identifier string
	Address    string
}

func (e TCPEntryPoint) Listen() (net.Listener, error) {
	return net.Listen("tcp", e.Address)
}

func (e TCPEntryPoint) Id() string {
	return e.Identifier
}
