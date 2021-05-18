package netplumbing

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
)

type Settings struct {
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

// settingsKey is the key used by context.WithValue/ctx.Value.
type settingsKey struct{}

// WithSettings returns a copy of the context using the provided Settings. This
// function will panic if passed a nil settings.
func WithSettings(ctx context.Context, settings *Settings) context.Context {
	if settings == nil {
		panic("oonet: WithSettings passed a nil pointer")
	}
	return context.WithValue(ctx, settingsKey{}, settings)
}

// ContextSettings returns the settings associated to the context. This function
// may return a nil Settings, if no Settings is saved into the context.
func ContextSettings(ctx context.Context) *Settings {
	settings, _ := ctx.Value(settingsKey{}).(*Settings)
	return settings
}
