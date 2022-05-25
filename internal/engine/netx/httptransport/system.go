package httptransport

import (
	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewSystemTransport creates a new "system" HTTP transport. That is a transport
// using the Go standard library with custom dialer and TLS dialer.
//
// Deprecation warning
//
// New code should use netxlite.NewHTTPTransport instead.
func NewSystemTransport(config Config) model.HTTPTransport {
	txp := oohttp.DefaultTransport.(*oohttp.Transport).Clone()
	txp.DialContext = config.Dialer.DialContext
	txp.DialTLSContext = config.TLSDialer.DialTLSContext
	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.MaxConnsPerHost = 1
	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true
	return &SystemTransportWrapper{&oohttp.StdlibTransport{Transport: txp}}
}

// SystemTransportWrapper adapts *http.Transport to have the .Network method
type SystemTransportWrapper struct {
	*oohttp.StdlibTransport
}

func (txp *SystemTransportWrapper) Network() string {
	return "tcp"
}

var _ model.HTTPTransport = &SystemTransportWrapper{}
