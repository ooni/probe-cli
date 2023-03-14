package dnsreport

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// defaultHTTPTransport is the default HTTP transport to use in this package. We want
// to use a transport featuring multiple parallel connections to the same host, a feature
// that is not implemented by netxlite. Hence, we adapt the default transport used by
// the net/http library to look like a [model.HTTPTransport].
var defaultHTTPTransport model.HTTPTransport = newHTTPTransport()

// defaultHTTPTransportDialContext is an adapter that returns the DialContext function
// given a specific [net.Dialer].
//
// Copied from Go 1.19.6's src/net/http/transport_default_other.go
//
// SPDX-License-Identifier: BSD-3-Clause
func defaultHTTPTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

// newHTTPTransport creates an instance of [model.HTTPTransport].
func newHTTPTransport() model.HTTPTransport {
	return &httpTransport{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: defaultHTTPTransportDialContext(&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}),
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// httpTransport implements [model.HTTPTransport].
type httpTransport struct {
	*http.Transport
}

// Network implements model.HTTPTransport
func (txp *httpTransport) Network() string {
	return "tcp"
}
