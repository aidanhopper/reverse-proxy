package engine

import "crypto/tls"

type TLSConfigHandler interface {
	HandleTLSConfig(*tls.ClientHelloInfo) (*tls.Config, error)
}

type TLSConfigHandlerFunc func(info *tls.ClientHelloInfo) (*tls.Config, error)

func (f TLSConfigHandlerFunc) HandleTLSConfig(info *tls.ClientHelloInfo) (*tls.Config, error) {
	return f(info)
}
