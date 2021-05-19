package netplumbing

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/bassosimone/quic-go/http3"
)

// Transport implements Transport.
type Transport struct {
	// RoundTripper is the underlying http.Transport. You need to
	// configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport.
	RoundTripper *http.Transport

	// HTTP3RoundTripper is the underlying http3.Transport. You need
	// to configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport.
	HTTP3RoundTripper *http3.RoundTripper
}

// NewTransport creates a new instance of Transport using a
// default underlying http.Transport.
func NewTransport() *Transport {
	txp := &Transport{}
	txp.RoundTripper = &http.Transport{
		Proxy:                 txp.proxy,
		DialContext:           txp.DialContext,
		DialTLSContext:        txp.DialTLSContext,
		TLSHandshakeTimeout:   txp.tlsHandshakeTimeout(),
		DisableCompression:    true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	txp.HTTP3RoundTripper = &http3.RoundTripper{
		DisableCompression: true,
		Dial:               txp.http3dial,
	}
	return txp
}

// DefaultTransport is the default implementation of Transport.
var DefaultTransport = NewTransport()

// proxy checks whether we need to use a proxy.
func (txp *Transport) proxy(req *http.Request) (*url.URL, error) {
	ctx := req.Context()
	// we know how to manage a socks5 proxy ourself, so in case the proxy
	// type is socks5, we use our own technique, which leads to better measurements
	if config := ContextConfig(ctx); config != nil && config.Proxy != nil &&
		config.Proxy.Scheme != "socks5" {
		log := txp.logger(ctx)
		log.Debugf("http: using proxy: %s", config.Proxy)
		return config.Proxy, nil
	}
	return nil, nil
}

// byteCounter returns the ByteCounter to use.
func (txp *Transport) byteCounter(ctx context.Context) ByteCounter {
	if config := ContextConfig(ctx); config != nil && config.ByteCounter != nil {
		return config.ByteCounter
	}
	return &noopByteCounter{}
}

// noopByteCounter is a no-op ByteCounter.
type noopByteCounter struct{}

// CountyBytesReceived increments the bytes received count.
func (*noopByteCounter) CountBytesReceived(count int) {}

// CountBytesSent increments the bytes sent count.
func (*noopByteCounter) CountBytesSent(count int) {}

// logger returns the configured logger or the DefaultLogger.
func (txp *Transport) logger(ctx context.Context) Logger {
	if config := ContextConfig(ctx); config != nil && config.Logger != nil {
		return config.Logger
	}
	return &quietLogger{}
}

// quietLogger is a logger that doesn't emit any message.
type quietLogger struct{}

// Debugf implements Logger.Debugf.
func (*quietLogger) Debugf(format string, v ...interface{}) {}

// Debug implements Logger.Debug.
func (*quietLogger) Debug(message string) {}

// tlsClientConfig returns the configured TLS client config or the default.
func (txp *Transport) tlsClientConfig(ctx context.Context) *tls.Config {
	if config := ContextConfig(ctx); config != nil && config.TLSClientConfig != nil {
		return config.TLSClientConfig.Clone()
	}
	return &tls.Config{}
}

// tlsHandshakeTimeout returns the TLS handshake timeout.
func (txp *Transport) tlsHandshakeTimeout() time.Duration {
	return 10 * time.Second
}

// CloseIdleConnections closes idle connections.
func (txp *Transport) CloseIdleConnections() {
	txp.RoundTripper.CloseIdleConnections()
	txp.HTTP3RoundTripper.Close()
}
