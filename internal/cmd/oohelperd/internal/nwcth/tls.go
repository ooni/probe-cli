package nwcth

import (
	"context"
	"crypto/tls"
	"net"
)

// QUICConfig configures the QUIC handshake check.
type TLSConfig struct {
	Conn     net.Conn
	Endpoint string
	Cfg      *tls.Config
}

// TLSDo performs the TLS check.
func TLSDo(ctx context.Context, config *TLSConfig) (*tls.Conn, *CtrlTLSMeasurement) {
	c := tls.Client(config.Conn, config.Cfg)
	err := c.Handshake()
	return c, &CtrlTLSMeasurement{
		Failure: newfailure(err),
	}
}
