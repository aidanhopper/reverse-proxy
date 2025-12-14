package proxy

import (
	"net"
)

type EntryPoint interface {
	Id() string
	Listen() (net.Listener, error)
}
