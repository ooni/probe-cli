package netplumbing

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
)

// ByteCounter counts bytes received and sent.
type ByteCounter interface {
	// CountyBytesReceived increments the bytes-received count.
	CountBytesReceived(count int)

	// CountBytesSent increments the bytes-sent count.
	CountBytesSent(count int)
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
