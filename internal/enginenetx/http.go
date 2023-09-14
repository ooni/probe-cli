package enginenetx

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/publicsuffix"
)

// HTTPTransport is the [model.HTTPTransport] used by the [*engine.Session].
type HTTPTransport struct {
	model.HTTPTransport
}

// NewHTTPClient is a convenience function for building a [model.HTTPClient] using
// this [*HTTPTransport] and correct cookies configuration.
func (txp *HTTPTransport) NewHTTPClient() *http.Client {
	// Note: cookiejar.New cannot fail, so we're using runtimex.Try1 here
	return &http.Client{
		Transport: txp,
		Jar: runtimex.Try1(cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})),
	}
}

// NewHTTPTransport creates a new [*HTTPTransport] for the engine. This client MUST NOT be
// used for measuring and implements engine-specific policies.
//
// Arguments:
//
// - counter is the [*bytecounter.Counter] to use.
//
// - logger is the [model.Logger] to use;
//
// - proxyURL is the OPTIONAL proxy URL;
//
// - resolver is the [model.Resolver] to use.
func NewHTTPTransport(
	counter *bytecounter.Counter,
	logger model.Logger,
	proxyURL *url.URL,
	resolver model.Resolver,
) *HTTPTransport {
	txp := netxlite.NewHTTPTransportWithLoggerResolverAndOptionalProxyURL(
		logger, resolver, proxyURL,
	)
	txp = bytecounter.WrapHTTPTransport(txp, counter)
	return &HTTPTransport{txp}
}
