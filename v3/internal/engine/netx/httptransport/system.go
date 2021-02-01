package httptransport

import (
	"net/http"
)

// NewSystemTransport creates a new "system" HTTP transport. That is a transport
// using the Go standard library with custom dialer and TLS dialer.
func NewSystemTransport(config Config) RoundTripper {
	txp := http.DefaultTransport.(*http.Transport).Clone()
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
	return txp
}

var _ RoundTripper = &http.Transport{}
