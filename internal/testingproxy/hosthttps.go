package testingproxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// WithHostNetworkHTTPWithTLSProxyAndURL returns a [TestCase] where:
//
// - we fetch a URL;
//
// - using the host network;
//
// - and an HTTPS proxy.
//
// Because this [TestCase] uses the host network, it does not run in -short mode.
func WithHostNetworkHTTPWithTLSProxyAndURL(URL string) TestCase {
	return &hostNetworkTestCaseWithHTTPWithTLS{
		TargetURL: URL,
	}
}

type hostNetworkTestCaseWithHTTPWithTLS struct {
	TargetURL string
}

var _ TestCase = &hostNetworkTestCaseWithHTTPWithTLS{}

// Name implements TestCase.
func (tc *hostNetworkTestCaseWithHTTPWithTLS) Name() string {
	return fmt.Sprintf("fetching %s using the host network and an HTTPS proxy", tc.TargetURL)
}

// Run implements TestCase.
func (tc *hostNetworkTestCaseWithHTTPWithTLS) Run(t *testing.T) {
	// create an instance of Netx where the underlying network is nil,
	// which means we're using the host's network
	netx := &netxlite.Netx{Underlying: nil}

	// create CA
	proxyCA := netem.MustNewCA()

	// create the proxy server using the host network
	proxyServer := testingx.MustNewHTTPServerTLS(
		testingx.NewHTTPProxyHandler(log.Log, netx),
		proxyCA,
		"proxy.local",
	)
	defer proxyServer.Close()

	// extend the default cert pool with the proxy's own CA
	pool := netxlite.NewMozillaCertPool()
	pool.AddCert(proxyServer.CACert)
	tlsConfig := &tls.Config{RootCAs: pool}

	// create an HTTP client configured to use the given proxy
	//
	// note how we use a dialer that asserts that we're using the proxy IP address
	// rather than the host address, so we're sure that we're using the proxy
	dialer := &dialerWithAssertions{
		ExpectAddress: "127.0.0.1",
		Dialer:        netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log)),
	}
	tlsDialer := netxlite.NewTLSDialerWithConfig(
		dialer, netxlite.NewTLSHandshakerStdlib(log.Log),
		tlsConfig,
	)
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer,
		netxlite.HTTPTransportOptionProxyURL(runtimex.Try1(url.Parse(proxyServer.URL))))
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()

	// get the homepage and assert we're getting a succesful response
	httpCheckResponse(t, client, tc.TargetURL)
}

// Short implements TestCase.
func (tc *hostNetworkTestCaseWithHTTPWithTLS) Short() bool {
	return false
}
