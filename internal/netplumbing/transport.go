package netplumbing

// This file contains the top-level implementation of Transport.

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/bassosimone/quic-go/http3"
)

// Transport allows you to perform network measurements and
// collect traces during the measurements.
//
// You configure a Transport using Config, which allows you
// to modify the way in which a Transport behaves.
//
// You can collect traces using a TraceHeader.
type Transport struct {
	// RoundTripper is the underlying http.Transport. You need to
	// configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport. Calling CloseIdleConnections
	// will close the idle connections on this transport.
	//
	// This is the round tripper used by default.
	RoundTripper *http.Transport

	// HTTP3RoundTripper is the underlying http3.Transport. You need
	// to configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport. Calling CloseIdleConnections
	// will close the idle connections on this transport.
	//
	// To use this round tripper, you need to configure it explicitly
	// inside the Config struct that you bind to a Context.
	HTTP3RoundTripper *http3.RoundTripper

	// OORoundTripper is the underlying OOHTTPTransport. You need
	// to configure this field. Otherwise, use NewTransport to obtain
	// a default configured Transport. Calling CloseIdleConnections
	// will close the idle connections on this transport.
	//
	// To use this round tripper, you need to configure it explicitly
	// inside the Config struct that you bind to a Context.
	//
	// Unlike RoundTripper, this round tripper is out-of-the-box
	// compatible with the UTLSHandshaker handshaker.
	OORoundTripper *OOHTTPTransport
}

// NewTransport creates a new instance of Transport and
// filling all the fields with reasonable defaults.
func NewTransport() *Transport {
	txp := &Transport{}
	txp.RoundTripper = &http.Transport{
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
	txp.OORoundTripper = &OOHTTPTransport{
		Transport: txp,
	}
	return txp
}

// DefaultTransport is the default Transport.
var DefaultTransport = NewTransport()

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
	txp.OORoundTripper.CloseIdleConnections()
}
