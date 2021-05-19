package netplumbing

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/bassosimone/quic-go"
)

// Config contains settings for the Transport. To pass configuration
// to a Transport, you need to create a Config and bind it to a context
// using the netplumbing.WithConfig function. The Transport will use
// the netplumbing.ContextConfig function to retrieve the Config.
type Config struct {
	// ByteCounter is the optional byte counter to use.
	ByteCounter ByteCounter

	// Connector is the optional connector to use.
	Connector Connector

	// HTTPHost allows to override the HTTP host header.
	HTTPHost string

	// HTTPTransport is the optional HTTP transport to use. The documented
	// way to force using HTTP3 is to override this field to point to the
	// HTTP3RoundTripper exported by the netplumbing.Transport.
	HTTPTransport http.RoundTripper

	// HTTPUserAgent allows to override the HTTP user agent.
	HTTPUserAgent string

	// Logger is the optional logger to use.
	Logger Logger

	// Proxy is the optional proxy URL.
	Proxy *url.URL

	// QUICConfig is the optional QUIC config.
	QUICConfig *quic.Config

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
	// QUICHandshake uses the local pconn to perform a QUIC handshake with the
	// remoteAddr using the settings in tlsConf and config.
	QUICHandshake(ctx context.Context, pconn net.PacketConn, remoteAddr net.Addr,
		tlsConf *tls.Config, config *quic.Config) (quic.EarlySession, error)
}

// QUICListener is a listener for QUIC.
type QUICListener interface {
	// QUICListen starts a listening UDP connection for QUIC.
	QUICListen(ctx context.Context) (net.PacketConn, error)
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
	// TLSHandshake performs the TLS handshake using the given tcpConn
	// and the settings contained into the config object.
	TLSHandshake(ctx context.Context, tcpConn net.Conn, config *tls.Config) (
		tlsConn net.Conn, state *tls.ConnectionState, err error)
}

// configKey is the key used by context.WithValue/ctx.Value.
type configKey struct{}

// WithConfig returns a copy of the context using the provided config. This
// function will panic if passed a nil config.
func WithConfig(ctx context.Context, config *Config) context.Context {
	if config == nil {
		panic("netplumbing: WithConfig passed a nil pointer")
	}
	return context.WithValue(ctx, configKey{}, config)
}

// ContextConfig returns the config associated to the context. This function
// may return a nil config, if no config is saved into the context.
func ContextConfig(ctx context.Context) *Config {
	config, _ := ctx.Value(configKey{}).(*Config)
	return config
}
