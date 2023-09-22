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

// Network is the network abstraction used by the OONI engine.
type Network struct {
	txp model.HTTPTransport
}

// HTTPTransport returns the [model.HTTPTransport] that the engine should use.
func (n *Network) HTTPTransport() model.HTTPTransport {
	return n.txp
}

// NewHTTPClient is a convenience function for building a [model.HTTPClient] using
// the underlying [model.HTTPTransport] and the correct cookies configuration.
func (n *Network) NewHTTPClient() *http.Client {
	// Note: cookiejar.New cannot fail, so we're using runtimex.Try1 here
	return &http.Client{
		Transport: n.txp,
		Jar: runtimex.Try1(cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})),
	}
}

// Close ensures that we close idle connections and persist statistics.
func (n *Network) Close() error {
	// nothing for now!
	return nil
}

// NewNetwork creates a new [*Network] for the engine. This network MUST NOT be
// used for measuring because it implements engine-specific policies.
//
// You MUST call the Close method when done using the network. This method ensures
// that (i) we close idle connections and (ii) persist statistics.
//
// Arguments:
//
// - counter is the [*bytecounter.Counter] to use.
//
// - kvStore is a [model.KeyValueStore] for persisting stats;
//
// - logger is the [model.Logger] to use;
//
// - proxyURL is the OPTIONAL proxy URL;
//
// - resolver is the [model.Resolver] to use.
//
// The presence of the proxyURL will cause this function to possibly build a
// network with different behavior with respect to circumvention. If there is
// an upstream proxy we're going to trust it is doing circumvention for us.
func NewNetwork(
	counter *bytecounter.Counter,
	kvStore model.KeyValueStore,
	logger model.Logger,
	proxyURL *url.URL,
	resolver model.Resolver,
) *Network {
	dialer := netxlite.NewDialerWithResolver(logger, resolver)
	handshaker := netxlite.NewTLSHandshakerStdlib(logger)
	tlsDialer := netxlite.NewTLSDialer(dialer, handshaker)
	txp := netxlite.NewHTTPTransportWithOptions(
		logger, dialer, tlsDialer,
		netxlite.HTTPTransportOptionDisableCompression(false),
		netxlite.HTTPTransportOptionProxyURL(proxyURL), // nil implies "no proxy"
	)
	txp = bytecounter.WrapHTTPTransport(txp, counter)
	return &Network{txp}
}
