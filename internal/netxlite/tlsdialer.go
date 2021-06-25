package netxlite

import (
	"context"
	"crypto/tls"
	"net"
)

// TLSDialer is the TLS dialer
type TLSDialer struct {
	// Config is the OPTIONAL tls config.
	Config *tls.Config

	// Dialer is the MANDATORY dialer.
	Dialer Dialer

	// TLSHandshaker is the MANDATORY TLS handshaker.
	TLSHandshaker TLSHandshaker
}

// DialTLSContext dials a TLS connection.
func (d *TLSDialer) DialTLSContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	config := d.config(host, port)
	tlsconn, _, err := d.TLSHandshaker.Handshake(ctx, conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return tlsconn, nil
}

// config creates a new config. If d.Config is nil, then we start
// from an empty config. Otherwise, we clone d.Config.
//
// We set the ServerName field if not already set.
//
// We set the ALPN if the port is 443 or 853, if not already set.
//
// We force using our root CA, unless it's already set.
func (d *TLSDialer) config(host, port string) *tls.Config {
	config := d.Config
	if config == nil {
		config = &tls.Config{}
	}
	config = config.Clone() // operate on a clone
	if config.ServerName == "" {
		config.ServerName = host
	}
	if len(config.NextProtos) <= 0 {
		switch port {
		case "443":
			config.NextProtos = []string{"h2", "http/1.1"}
		case "853":
			config.NextProtos = []string{"dot"}
		}
	}
	if config.RootCAs == nil {
		config.RootCAs = NewDefaultCertPool()
	}
	return config
}
