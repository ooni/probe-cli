package netxlite

import (
	"net/url"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPTransportOption is an initialization option for [NewHTTPTransport].
type HTTPTransportOption func(txp *oohttp.Transport)

// NewHTTPTransport is the high-level factory to create a [model.HTTPTransport] using
// github.com/ooni/oohttp as the HTTP library with HTTP/1.1 and HTTP2 support.
//
// This transport is suitable for HTTP2 and HTTP/1.1 using any TLS
// library, including, e.g., github.com/ooni/oocrypto.
//
// This factory clones the default github.com/ooni/oohttp transport and
// configures the provided dialer and TLS dialer by setting the .DialContext
// and .DialTLSContext fields of the transport. We also wrap the provided
// dialers to address https://github.com/ooni/probe/issues/1609.
//
// Apart from that, the only non-default options set by this factory are these:
//
// 1. the .Proxy field is set to nil, so by default we DO NOT honour the
// HTTP_PROXY and HTTPS_PROXY environment variables, which is required if
// we want to use this code for measuring;
//
// 2. the .ForceAttemptHTTP2 field is set to true.
//
// The returned transport supports logging and error wrapping because
// internally this function calls [WrapHTTPTransport] before we return.
//
// This factory is the RECOMMENDED way of creating a [model.HTTPTransport].
func NewHTTPTransportWithOptions(logger model.Logger,
	dialer model.Dialer, tlsDialer model.TLSDialer, options ...HTTPTransportOption) model.HTTPTransport {
	// Using oohttp to support any TLS library.
	txp := oohttp.DefaultTransport.(*oohttp.Transport).Clone()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.DialContext = dialer.DialContext
	tlsDialer = &httpTLSDialerWithReadTimeout{tlsDialer}
	txp.DialTLSContext = tlsDialer.DialTLSContext

	// As documented, disable proxies and force HTTP/2
	txp.Proxy = nil
	txp.ForceAttemptHTTP2 = true

	// Apply all the required options
	for _, option := range options {
		option(txp)
	}

	// Return a fully wrapped HTTP transport
	return WrapHTTPTransport(logger, &httpTransportConnectionsCloser{
		HTTPTransport: &httpTransportStdlib{&oohttp.StdlibTransport{Transport: txp}},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	})
}

// HTTPTransportOptionProxyURL configures the transport to use the given proxyURL
// or disables proxying (already the default) if the proxyURL is nil.
func HTTPTransportOptionProxyURL(proxyURL *url.URL) HTTPTransportOption {
	return func(txp *oohttp.Transport) {
		txp.Proxy = func(r *oohttp.Request) (*url.URL, error) {
			// "If Proxy is nil or returns a nil *URL, no proxy is used."
			return proxyURL, nil
		}
	}
}

// HTTPTransportOptionMaxConnsPerHost configures the .MaxConnPerHosts field, which
// otherwise uses the default set in github.com/ooni/oohttp.
func HTTPTransportOptionMaxConnsPerHost(value int) HTTPTransportOption {
	return func(txp *oohttp.Transport) {
		txp.MaxConnsPerHost = value
	}
}

// HTTPTransportOptionDisableCompression configures the .DisableCompression field, which
// otherwise uses the default set in github.com/ooni/oohttp, i.e., false.
func HTTPTransportOptionDisableCompression(value bool) HTTPTransportOption {
	return func(txp *oohttp.Transport) {
		txp.DisableCompression = value
	}
}
