package testingproxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// WithNetemHTTPWithTLSProxyAndURL returns a [TestCase] where:
//
// - we fetch a URL;
//
// - using the github.com/ooni.netem;
//
// - and an HTTPS proxy.
//
// Because this [TestCase] uses netem, it also runs in -short mode.
func WithNetemHTTPWithTLSProxyAndURL(URL string) TestCase {
	return &netemTestCaseWithHTTPWithTLS{
		TargetURL: URL,
	}
}

type netemTestCaseWithHTTPWithTLS struct {
	TargetURL string
}

var _ TestCase = &netemTestCaseWithHTTPWithTLS{}

// Name implements TestCase.
func (tc *netemTestCaseWithHTTPWithTLS) Name() string {
	return fmt.Sprintf("fetching %s using netem and an HTTPS proxy", tc.TargetURL)
}

// Run implements TestCase.
func (tc *netemTestCaseWithHTTPWithTLS) Run(t *testing.T) {
	topology := runtimex.Try1(netem.NewStarTopology(log.Log))
	defer topology.Close()

	const (
		wwwIPAddr    = "93.184.216.34"
		proxyIPAddr  = "10.0.0.1"
		clientIPAddr = "10.0.0.2"
	)

	// create:
	//
	// - a www stack modeling www.example.com
	//
	// - a proxy stack
	//
	// - a client stack
	//
	// Note that www.example.com's IP address is also the resolver used by everyone
	wwwStack := runtimex.Try1(topology.AddHost(wwwIPAddr, wwwIPAddr, &netem.LinkConfig{}))
	proxyStack := runtimex.Try1(topology.AddHost(proxyIPAddr, wwwIPAddr, &netem.LinkConfig{}))
	clientStack := runtimex.Try1(topology.AddHost(clientIPAddr, wwwIPAddr, &netem.LinkConfig{}))

	// configure the wwwStack as the DNS resolver with proper configuration
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord("www.example.com.", "", wwwIPAddr)
	dnsServer := runtimex.Try1(netem.NewDNSServer(log.Log, wwwStack, wwwIPAddr, dnsConfig))
	defer dnsServer.Close()

	// configure the wwwStack to respond to HTTP requests on port 80
	wwwServer80 := testingx.MustNewHTTPServerEx(
		&net.TCPAddr{IP: net.ParseIP(wwwIPAddr), Port: 80},
		wwwStack,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Bonsoir, Elliot!\r\n"))
		}),
	)
	defer wwwServer80.Close()

	// configure the wwwStack to respond to HTTPS requests on port 443
	wwwServer443 := testingx.MustNewHTTPServerTLSEx(
		&net.TCPAddr{IP: net.ParseIP(wwwIPAddr), Port: 443},
		wwwStack,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Bonsoir, Elliot!\r\n"))
		}),
		wwwStack,
	)
	defer wwwServer443.Close()

	// configure the proxyStack to implement the HTTP proxy on port 8080
	proxyServer := testingx.MustNewHTTPServerTLSEx(
		&net.TCPAddr{IP: net.ParseIP(proxyIPAddr), Port: 80443},
		proxyStack,
		testingx.NewHTTPProxyHandler(log.Log, &netxlite.Netx{
			Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: proxyStack}}),
		proxyStack,
	)
	defer proxyServer.Close()

	// crete the netx instance for the client
	netx := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}}

	log.SetLevel(log.DebugLevel)

	/*
		 ,ggg,      gg      ,gg,ggggggggggggggg ,gggggggggggggg
		dP""Y8a     88     ,8PdP""""""88"""""""dP""""""88""""""
		Yb, `88     88     d8'Yb,_    88       Yb,_    88
		 `"  88     88     88  `""    88        `""    88
		     88     88     88         88            ggg88gggg
		     88     88     88         88               88   8
		     88     88     88         88               88
		     Y8    ,88,    8P   gg,   88         gg,   88
		      Yb,,d8""8b,,dP     "Yb,,8P          "Yb,,8P
		       "88"    "88"        "Y8P'            "Y8P'


		Not necessarily wrong, but certainly I did not expect this! When we are
		using an HTTPS proxy, the stdlib/oohttp *Transport uses the DialTLSContext for
		dialing with the proxy but then uses its own TLS for handshaking over
		the TLS connection with the proxy.

		So, the naive implementation of this test case fails with an X.509
		certificate error when we're using netem, because we're not using the
		overriden DialTLSContext anymore, and the *Transport does not
		otherwise know about the root CA used by netem.

		The current fix is to use netxlite.HTTPTransportOptionTLSClientConfig
		below. However, I'm now wondering if we're using the right default.
	*/

	// create an HTTP client configured to use the given proxy
	//
	// note how we use a dialer that asserts that we're using the proxy IP address
	// rather than the host address, so we're sure that we're using the proxy
	dialer := &dialerWithAssertions{
		ExpectAddress: proxyIPAddr,
		Dialer:        netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log)),
	}
	tlsDialer := netxlite.NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(log.Log))
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer,
		netxlite.HTTPTransportOptionProxyURL(runtimex.Try1(url.Parse(proxyServer.URL))),

		// see above WTF comment
		netxlite.HTTPTransportOptionTLSClientConfig(&tls.Config{
			RootCAs: runtimex.Try1(clientStack.DefaultCertPool()),
		}),
	)
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()

	// get the homepage and assert we're getting a succesful response
	httpCheckResponse(t, client, tc.TargetURL)
}

// Short implements TestCase.
func (tc *netemTestCaseWithHTTPWithTLS) Short() bool {
	return true
}
