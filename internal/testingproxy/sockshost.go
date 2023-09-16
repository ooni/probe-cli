package testingproxy

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingsocks5"
)

// WithHostNetworkSOCKSProxyAndURL returns a [TestCase] where:
//
// - we fetch a URL;
//
// - using the host network;
//
// - and an HTTP proxy.
//
// Because this [TestCase] uses the host network, it does not run in -short mode.
func WithHostNetworkSOCKSProxyAndURL(URL string) TestCase {
	return &hostNetworkTestCaseWithSOCKS{
		TargetURL: URL,
	}
}

type hostNetworkTestCaseWithSOCKS struct {
	TargetURL string
}

var _ TestCase = &hostNetworkTestCaseWithSOCKS{}

// Name implements TestCase.
func (tc *hostNetworkTestCaseWithSOCKS) Name() string {
	return fmt.Sprintf("fetching %s using the host network and an HTTP proxy", tc.TargetURL)
}

// Run implements TestCase.
func (tc *hostNetworkTestCaseWithSOCKS) Run(t *testing.T) {
	// create an instance of Netx where the underlying network is nil,
	// which means we're using the host's network
	netx := &netxlite.Netx{Underlying: nil}

	// create the proxy server using the host network
	endpoint := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	proxyServer := testingsocks5.MustNewServer(log.Log, netx, endpoint)
	defer proxyServer.Close()

	//log.SetLevel(log.DebugLevel)

	// create an HTTP client configured to use the given proxy
	//
	// note how we use a dialer that asserts that we're using the proxy IP address
	// rather than the host address, so we're sure that we're using the proxy
	dialer := &dialerWithAssertions{
		ExpectAddress: "127.0.0.1",
		Dialer:        netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log)),
	}
	tlsDialer := netxlite.NewTLSDialer(dialer, netxlite.NewTLSHandshakerStdlib(log.Log))
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer,
		netxlite.HTTPTransportOptionProxyURL(proxyServer.URL()))
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()

	// get the homepage and assert we're getting a succesful response
	httpCheckResponse(t, client, tc.TargetURL)
}

// Short implements TestCase.
func (tc *hostNetworkTestCaseWithSOCKS) Short() bool {
	return false
}
