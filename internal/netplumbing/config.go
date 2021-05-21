package netplumbing

// This file contains everything related to the Config struct.

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
	// ByteCounter is the optional byte counter to use. If you
	// configure this field, then we will call its methods every
	// time a read/read_from or write/write_to completes. This
	// is the documented way to track the bytes sent or received
	// and/or to keep track of bandwidth usage.
	ByteCounter ByteCounter

	// Connector is the optional connector to use. Setting this
	// field means that every piece of code connecting a TCP/UDP
	// socket will call its DialContext method to do that. The
	// address argument to the method will always be an IP address.
	Connector Connector

	// HTTPHost allows to override the HTTP host header. If you
	// do that, we will connect() to the IP/domain in the URL.Host
	// but we'll send this field as part of the HTTP headers.
	//
	// This is mostly useful to detect censorship based on the
	// host header with unencrypted HTTP/1.1 flows.
	HTTPHost string

	// HTTPTransport is the optional HTTP transport to use.
	//
	// The documented way to force using HTTP3 is to override this
	// field to point to Transport.HTTP3RoundTripper.
	//
	// The documented way of using the OONI replacement for the
	// stdlib transport (which is compatible with UTLS) is to
	// overide this field to point to Transport.OORoundTripper.
	HTTPTransport http.RoundTripper

	// HTTPUserAgent allows to override the HTTP user agent. If not
	// set then we will use DefaultUserAgent.
	HTTPUserAgent string

	// Logger is the optional logger to use. This interface
	// is compatible with github.com/apex/log's logger.
	Logger Logger

	// Proxy is the optional proxy URL. We support "http" and
	// "socks5" proxies with optional username and password.
	Proxy *url.URL

	// QUICConfig is the optional QUIC config. If not set, then
	// we use an empty QUIC config.
	QUICConfig *quic.Config

	// QUICHandshaker is the optional QUIC handshaker to use. If
	// set, then we'll use it for QUIC handshakes.
	QUICHandshaker QUICHandshaker

	// QUICListener is the optional listener for QUIC to use. If set,
	// we'll use it to create QUIC UDP listening sockets.
	QUICListener QUICListener

	// Resolver is the optional resolver to use. If not set, then
	// we'll use the standard library's resolver.
	//
	// The documented way to force a custom resolver is to create
	// an instance of DNSResolver using NewDNSResolver and overriding
	// this Config field to point to such an instance.
	Resolver Resolver

	// TLSClientConfig is the optional TLS config to use. If not
	// set, then we'll use an empty config for TLS and QUIC.
	TLSClientConfig *tls.Config

	// TLSHandshaker is the optional TLS handshaker to use. If
	// set, we'll use it instead of the stdlib.
	//
	// The documented way to use UTLS is to create an instance
	// of the UTLSHandshaker and point it to this field.
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
