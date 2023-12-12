package netxlite

import (
	"crypto/tls"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/feature/oohttpfeat"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPTransportOption is an initialization option for [NewHTTPTransport].
type HTTPTransportOption func(txp *oohttpfeat.HTTPTransport)

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
// 2. the .ForceAttemptHTTP2 field is set to true;
//
// 3. the .DisableCompression field is set to true, again required if we
// want to use this code for measuring, and we should make sure the defaults
// we're using are suitable for measuring, since the impact of making a
// mistake in measuring code is a data quality issue 😅.
//
// The returned transport supports logging and error wrapping because
// internally this function calls [WrapHTTPTransport] before we return.
//
// This factory is the RECOMMENDED way of creating a [model.HTTPTransport].
func NewHTTPTransportWithOptions(logger model.Logger,
	dialer model.Dialer, tlsDialer model.TLSDialer, options ...HTTPTransportOption) model.HTTPTransport {
	// Using oohttp to support any TLS library iff it's possible to do so, otherwise we
	// are going to use the standard library w/o using HTTP/2 support.
	txp := oohttpfeat.NewHTTPTransport()

	// This wrapping ensures that we always have a timeout when we
	// are using HTTP; see https://github.com/ooni/probe/issues/1609.
	dialer = &httpDialerWithReadTimeout{dialer}
	txp.SetDialContext(dialer.DialContext)
	tlsDialer = &httpTLSDialerWithReadTimeout{tlsDialer}
	txp.SetDialTLSContext(tlsDialer.DialTLSContext)

	// As documented, disable proxies and force HTTP/2
	txp.SetDisableCompression(true)
	txp.SetProxy(nil)

	// We now rely on feature/oohttpfeat to decide whether it's
	// possible for us to actually enable HTTP/2.
	//
	//	txp.ForceAttemptHTTP2 = true
	//
	// Please, keep this comment until at least 3.21 because it provides
	// an historical documentation of how we changed the codebase.

	// Apply all the required options
	for _, option := range options {
		option(txp)
	}

	// Return a fully wrapped HTTP transport
	return WrapHTTPTransport(logger, &httpTransportConnectionsCloser{
		HTTPTransport: &httpTransportStdlib{txp},
		Dialer:        dialer,
		TLSDialer:     tlsDialer,
	})
}

// HTTPTransportOptionProxyURL configures the transport to use the given proxyURL
// or disables proxying (already the default) if the proxyURL is nil.
func HTTPTransportOptionProxyURL(proxyURL *url.URL) HTTPTransportOption {
	return func(txp *oohttpfeat.HTTPTransport) {
		txp.SetProxy(func(r *oohttpfeat.HTTPRequest) (*url.URL, error) {
			// "If Proxy is nil or returns a nil *URL, no proxy is used."
			return proxyURL, nil
		})
	}
}

// HTTPTransportOptionMaxConnsPerHost configures the .MaxConnPerHosts field, which
// otherwise uses the default set in github.com/ooni/oohttp.
func HTTPTransportOptionMaxConnsPerHost(value int) HTTPTransportOption {
	return func(txp *oohttpfeat.HTTPTransport) {
		txp.SetMaxConnsPerHost(value)
	}
}

// HTTPTransportOptionDisableCompression configures the .DisableCompression field, which
// otherwise is set to true, so that this code is ready for measuring out of the box.
func HTTPTransportOptionDisableCompression(value bool) HTTPTransportOption {
	return func(txp *oohttpfeat.HTTPTransport) {
		txp.SetDisableCompression(value)
	}
}

// HTTPTransportOptionTLSClientConfig configures the .TLSClientConfig field,
// which otherwise is nil, to imply using the default config.
//
// TODO(https://github.com/ooni/probe/issues/2536): using the default config breaks
// tests using netem and this option is the workaround we're using to address
// this limitation. Future releases MIGHT use a different technique and, as such,
// we MAY remove this option when we don't need it anymore.
func HTTPTransportOptionTLSClientConfig(config *tls.Config) HTTPTransportOption {
	return func(txp *oohttpfeat.HTTPTransport) {
		txp.SetTLSClientConfig(config)
	}
}
