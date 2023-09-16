package testingsocks5_test

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingproxy"
	"github.com/ooni/probe-cli/v3/internal/testingsocks5"
)

func TestNetem(t *testing.T) {
	for _, testCase := range testingproxy.SOCKSTestCases {
		t.Run(testCase.Name(), func(t *testing.T) {
			short := testCase.Short()
			if !short && testing.Short() {
				t.Skip("skip test in short mode")
			}
			testCase.Run(t)
		})
	}
}

func TestNetemDialFailure(t *testing.T) {
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

	// configure the proxyStack to implement the SOCKS proxy on port 9050
	proxyServer := testingsocks5.MustNewServer(
		log.Log,
		&netxlite.Netx{
			Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: proxyStack}},
		&net.TCPAddr{IP: net.ParseIP(proxyIPAddr), Port: 9050},
	)
	defer proxyServer.Close()

	// create the netx instance for the client
	netx := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}}

	log.SetLevel(log.DebugLevel)

	// create an HTTP client configured to use the given proxy
	dialer := netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log))
	tlsDialer := netxlite.NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(log.Log))
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer,
		netxlite.HTTPTransportOptionProxyURL(proxyServer.URL()),

		// TODO(https://github.com/ooni/probe/issues/2536)
		netxlite.HTTPTransportOptionTLSClientConfig(&tls.Config{
			RootCAs: runtimex.Try1(clientStack.DefaultCertPool()),
		}),
	)
	client := &http.Client{Transport: txp}
	defer client.CloseIdleConnections()

	// because the TCP/IP stack exists but we're not listening, we should get an error (the
	// SOCKS5 library has been simplified to always return "host unreachabile")
	resp, err := client.Get("https://www.example.com/")
	if err == nil || !strings.HasSuffix(err.Error(), "host unreachable") {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("expected nil resp")
	}
}
