package proxy

import (
	"errors"
	"net"
)

type Transport string

const (
	TransportTCP         Transport = "tcp"
	TransportUDP         Transport = "udp"
	TransportUnix        Transport = "unix"
	TransportUnsupported Transport = "unsupported"
)

func GetTransport(addr net.Addr) (Transport, error) {
	switch addr.Network() {
	case "tcp", "tcp4", "tcp6":
		return TransportTCP, nil
	case "udp", "udp4", "udp6":
		return TransportUDP, nil
	case "unix":
		return TransportUnix, nil
	default:
		return TransportUnsupported, errors.New("Unsupported transport type")
	}
}
