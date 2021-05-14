// Package oonet implements networking primitives.
package oonet

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Logger implements logging. This interface is compatible with
// the github.com/apex/log logging interface.
type Logger interface {
	// Debug emits a debugging message.
	Debug(message string)

	// Debugf formats and emits a debugging message.
	Debugf(format string, v ...interface{})

	// Info emits an informational message.
	Info(message string)

	// Infof formats and emits an informational message.
	Infof(format string, v ...interface{})

	// Warn emits a warning message.
	Warn(message string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// QuietLogger is a logger that doesn't emit any message.
type QuietLogger struct{}

// Debug implements Logger.Debug.
func (*QuietLogger) Debug(message string) {}

// Debugf implements Logger.Debugf.
func (*QuietLogger) Debugf(format string, v ...interface{}) {}

// Info implements Logger.Info.
func (*QuietLogger) Info(message string) {}

// Infof implements Logger.Infof.
func (*QuietLogger) Infof(format string, v ...interface{}) {}

// Warn implements Logger.Warn.
func (*QuietLogger) Warn(message string) {}

// Warnf implements Logger.Warnf.
func (*QuietLogger) Warnf(format string, v ...interface{}) {}

// DefaultLogger is the default logger.
var DefaultLogger = &QuietLogger{}

// TLSHandshakeResult contains the result of a TLS handshake.
type TLSHandshakeResult struct {
	// Conn is the TLS connection. You OWN this connection and MUST call
	// its Close method when done using it.
	Conn net.Conn

	// State is the TLS connection state.
	State *tls.ConnectionState
}

// Overrides contains overrides for Transport. You configure these overrides
// using the WithOverrides function on a specific context. Given a context, a
// Transport will use ContextOverrides to get the Overrides struct. If the
// Overrides struct is not nil, the Transport will use any non-nil function
// inside it instead of the default implementation.
type Overrides struct {
	// DialContext overrides Transport.DefaultDialContext.
	DialContext func(ctx context.Context, network string, addr string) (net.Conn, error)

	// DialTLSContext overrides Transport.DefaultDialTLSContext.
	DialTLSContext func(ctx context.Context, network string, addr string) (net.Conn, error)

	// LookupHost overrides Transport.DefaultLookupHost.
	LookupHost func(ctx context.Context, domain string) ([]string, error)

	// RoundTrip overrides Transport.DefaultRoundTrip.
	RoundTrip func(req *http.Request) (*http.Response, error)

	// TLSHandshake overrides Transport.DefaultTLSHandshake.
	TLSHandshake func(ctx context.Context, conn net.Conn, config *tls.Config) (
		*TLSHandshakeResult, error)
}

// overridesKey is the key used by context.WithValue/ctx.Value.
type overridesKey struct{}

// WithOverrides returns a copy of the context using the provided Overrides. This
// function will panic if passed a nil overrides.
func WithOverrides(ctx context.Context, overrides *Overrides) context.Context {
	if overrides == nil {
		panic("oonet: WithOverrides passed a nil pointer")
	}
	return context.WithValue(ctx, overridesKey{}, overrides)
}

// ContextOverrides returns the overrides associated to the context. This function
// may return a nil Overrides, if no Overrides is saved into the context.
func ContextOverrides(ctx context.Context) *Overrides {
	overrides, _ := ctx.Value(overridesKey{}).(*Overrides)
	return overrides
}

// Transport is an HTTP transport, a DNS resolver, and a dialer for cleartext
// and TLS connections. You MUST NOT directly modify any field of this data
// structure after initialization. Doing that would most likely cause data races.
type Transport struct {
	// Logger is the logger to use. If nil, we will instead use a logger
	// that does not emit any logging message.
	Logger Logger

	// Proxy specifies which proxy to use, if any. This setting WILL NOT have
	// any effect in the two following cases:
	//
	// 1. there is a custom override.RoundTrip override;
	//
	// 2. you call DialContext or DialTLSContext.
	//
	// A future version of this implementation may fix these limitations. We will
	// return ErrProxyNotImplemented in the latter case.
	Proxy func(*http.Request) (*url.URL, error)

	// TLSClientConfig contains the default tls.Config. If nil, we will create
	// a new tls.Config and fill it. Note that, in particular, by default we
	// will set the ALPN to {"h2", "http/1.1"} if the port is "443".
	TLSClientConfig *tls.Config

	// mu provides mutual exclusion.
	mu sync.Mutex

	// httpTransport is the underlying http.Transport. If nil, we will create a
	// suitable http.Transport the first time we need it.
	httpTransport http.RoundTripper
}

// logger returns the configured Logger or the default one.
func (txp *Transport) logger() Logger {
	if txp.Logger != nil {
		return txp.Logger
	}
	return DefaultLogger
}

// tlsClientConfig returns the configured TLS client config or the default.
func (txp *Transport) tlsClientConfig() *tls.Config {
	if txp.TLSClientConfig != nil {
		return txp.TLSClientConfig.Clone()
	}
	return &tls.Config{}
}

// tlsHandshakeTimeout returns the TLS handshake timeout.
func (txp *Transport) tlsHandshakeTimeout() time.Duration {
	return 10 * time.Second
}

// TODO(bassosimone): I suppose we want httpx here as
// methods of an http client so we don't need to force
// people to remember to read bodies in the right way
// where we honour the context?

// getOrCreateTransport creates (if needed) and then returns the
// internal httpTransport used by the Transport. This function will
// lock and unlock the underlying `mu` mutex.
func (txp *Transport) getOrCreateTransport() http.RoundTripper {
	defer txp.mu.Unlock()
	txp.mu.Lock()
	if txp.httpTransport == nil {
		txp.httpTransport = &http.Transport{
			Proxy:                 txp.Proxy,
			DialContext:           txp.httpDialContext,
			DialTLSContext:        txp.httpDialTLSContext,
			TLSClientConfig:       txp.TLSClientConfig,
			TLSHandshakeTimeout:   txp.tlsHandshakeTimeout(),
			DisableCompression:    true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		}
	}
	return txp.httpTransport
}

// CloseIdleConnections closes idle connections (if any).
func (txp *Transport) CloseIdleConnections() {
	type idleCloser interface {
		CloseIdleConnections()
	}
	if ic, ok := txp.getOrCreateTransport().(idleCloser); ok {
		ic.CloseIdleConnections()
	}
}
