package nwcth

import (
	"context"
	"crypto/tls"
	"net"
)

// TLSConfig configures the TLS handshake check.
type TLSConfig struct {
	Conn     net.Conn
	Endpoint string
	Cfg      *tls.Config
}

// TLSDo performs the TLS check.
func TLSDo(ctx context.Context, config *TLSConfig) (*tls.Conn, *TLSHandshakeMeasurement) {
	c := tls.Client(config.Conn, config.Cfg)
	err := c.Handshake()
	return c, &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
}
