package proxy

import "crypto/tls"

type TLSConfigCompiler interface {
	Compile(*BufferedConn) (*tls.Config, error)
}
