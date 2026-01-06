package engine

type Protocol string

const (
	ProtoHTTP Protocol = "http"
	ProtoTCP  Protocol = "tcp"
	ProtoUDP  Protocol = "udp"
	ProtoUnix Protocol = "unix"
)
