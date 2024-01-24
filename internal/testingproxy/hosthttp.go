package testingproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// WithHostNetworkHTTPProxyAndURL returns a [TestCase] where:
//
// - we fetch a URL;
//
// - using the host network;
//
// - and an HTTP proxy.
//
// Because this [TestCase] uses the host network, it does not run in -short mode.
func WithHostNetworkHTTPProxyAndURL(URL string) TestCase {
	return &hostNetworkTestCaseWithHTTP{
		TargetURL: URL,
	}
}

type hostNetworkTestCaseWithHTTP struct {
	TargetURL string
}

var _ TestCase = &hostNetworkTestCaseWithHTTP{}

// Name implements TestCase.
func (tc *hostNetworkTestCaseWithHTTP) Name() string {
	return fmt.Sprintf("fetching %s using the host network and an HTTP proxy", tc.TargetURL)
}

// Run implements TestCase.
func (tc *hostNetworkTestCaseWithHTTP) Run(t *testing.T) {
	// create an instance of Netx where the underlying network is nil,
	// which means we're using the host's network
	netx := &netxlite.Netx{Underlying: nil}

	// create the proxy server using the host network
	proxyServer := testingx.MustNewHTTPServer(testingx.NewHTTPProxyHandler(log.Log, netx))
	defer proxyServer.Close()

	// create an HTTP client configured to use the given proxy
	//
	// note how we use a dialer that asserts that we're using the proxy IP address
	// rather than the host address, so we're sure that we're using the proxy
	dialer := &dialerWithAssertions{
		ExpectAddress: "127.0.0.1",
		Dialer:        netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log)),
	}
	tlsDialer := netxlite.NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(log.Log))
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer,
		netxlite.HTTPTransportOptionProxyURL(runtimex.Try1(url.Parse(proxyServer.URL))))
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()

	// get the homepage and assert we're getting a succesful response
	httpCheckResponse(t, client, tc.TargetURL)
}

// Short implements TestCase.
func (tc *hostNetworkTestCaseWithHTTP) Short() bool {
	return false
}
