package netplumbing

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/bassosimone/quic-go"
)

// ByteCounter counts bytes received and sent.
type ByteCounter interface {
	// CountyBytesReceived increments the bytes-received count.
	CountBytesReceived(count int)

	// CountBytesSent increments the bytes-sent count.
	CountBytesSent(count int)
}

// Connector creates new network connections.
type Connector interface {
	// DialContext establishes a new network connection.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Logger formats and emits log messages.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Debug emits a debug message.
	Debug(message string)
}

// QUICHandshaker performs the QUIC handshake.
type QUICHandshaker interface {
	QUICHandshake(ctx context.Context, pconn net.PacketConn, remoteAddr net.Addr,
		tlsConf *tls.Config, config *quic.Config) (quic.EarlySession, error)
}

// QUICListener is a listener for QUIC.
type QUICListener interface {
	// QUICListen starts a listening UDP connection for QUIC.
	QUICListen(ctx context.Context) (*net.UDPConn, error)
}

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost maps a domain name to IP addresses. If domain is an IP
	// address, this function returns a list containing such IP address
	// as the unique list element, and no error (like getaddrinfo).
	LookupHost(ctx context.Context, domain string) (addrs []string, err error)
}

// TLSHandshaker performs a TLS handshake.
type TLSHandshaker interface {
	// TLSHandshake performs the TLS handshake.
	TLSHandshake(ctx context.Context, tcpConn net.Conn, config *tls.Config) (
		tlsConn net.Conn, state *tls.ConnectionState, err error)
}

// Config contains settings you can configure using WithConfig.
type Config struct {
	// ByteCounter is the optional byte counter to use.
	ByteCounter ByteCounter

	// Connector is the optional connector to use.
	Connector Connector

	// HTTPTransport is the optional HTTP transport to use.
	HTTPTransport http.RoundTripper

	// Logger is the optional logger to use.
	Logger Logger

	// Proxy is the proxy URL.
	Proxy *url.URL

	// QUICHandshaker is the optional QUIC handshaker to use.
	QUICHandshaker QUICHandshaker

	// QUICListener is the optional listener for QUIC to use.
	QUICListener QUICListener

	// Resolver is the optional resolver to use.
	Resolver Resolver

	// TLSClientConfig is the optional TLS config to use.
	TLSClientConfig *tls.Config

	// TLSHandshaker is the optional TLS handshaker to use.
	TLSHandshaker TLSHandshaker
}

// configKey is the key used by context.WithValue/ctx.Value.
type configKey struct{}

// WithConfig returns a copy of the context using the provided config. This
// function will panic if passed a nil config.
func WithConfig(ctx context.Context, config *Config) context.Context {
	if config == nil {
		panic("oonet: WithConfig passed a nil pointer")
	}
	return context.WithValue(ctx, configKey{}, config)
}

// ContextConfig returns the config associated to the context. This function
// may return a nil config, if no config is saved into the context.
func ContextConfig(ctx context.Context) *Config {
	config, _ := ctx.Value(configKey{}).(*Config)
	return config
}
