package testingx_test

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestTLSSNIProxy(t *testing.T) {
	// testcase is a test case run by this function
	type testcase struct {
		name      string
		construct func() (*testingx.TLSSNIProxy, *netxlite.Netx, []io.Closer)
		short     bool
	}

	testcases := []testcase{{
		name: "when using the real network",
		construct: func() (*testingx.TLSSNIProxy, *netxlite.Netx, []io.Closer) {
			var closers []io.Closer

			netxProxy := &netxlite.Netx{
				Underlying: nil, // use the network
			}
			tcpAddr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}

			proxy := testingx.MustNewTLSSNIProxyEx(log.Log, netxProxy, tcpAddr)
			closers = append(closers, proxy)

			netxClient := &netxlite.Netx{
				Underlying: nil, // use the network
			}

			return proxy, netxClient, closers
		},
		short: false,
	}, {
		name: "when using netem",
		construct: func() (*testingx.TLSSNIProxy, *netxlite.Netx, []io.Closer) {
			var closers []io.Closer

			topology := runtimex.Try1(netem.NewStarTopology(log.Log))
			closers = append(closers, topology)

			wwwStack := runtimex.Try1(topology.AddHost("142.251.209.14", "142.251.209.14", &netem.LinkConfig{}))
			proxyStack := runtimex.Try1(topology.AddHost("10.0.0.1", "142.251.209.14", &netem.LinkConfig{}))
			clientStack := runtimex.Try1(topology.AddHost("10.0.0.2", "142.251.209.14", &netem.LinkConfig{}))

			dnsConfig := netem.NewDNSConfig()
			dnsConfig.AddRecord("www.google.com", "", "142.251.209.14")
			dnsServer := runtimex.Try1(netem.NewDNSServer(log.Log, wwwStack, "142.251.209.14", dnsConfig))
			closers = append(closers, dnsServer)

			wwwServer := testingx.MustNewHTTPServerTLSEx(
				&net.TCPAddr{IP: net.IPv4(142, 251, 209, 14), Port: 443},
				wwwStack,
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Bonsoir, Elliot!"))
				}),
				wwwStack,
			)
			closers = append(closers, wwwServer)

			proxy := testingx.MustNewTLSSNIProxyEx(
				log.Log,
				&netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: proxyStack}},
				&net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 443},
			)
			closers = append(closers, proxy)

			clientNet := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}}
			return proxy, clientNet, closers
		},
		short: true,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.short && testing.Short() {
				t.Skip("skip test in short mode")
			}

			proxy, clientNet, closers := tc.construct()
			defer func() {
				for _, closer := range closers {
					closer.Close()
				}
			}()

			//log.SetLevel(log.DebugLevel)

			tlsConfig := &tls.Config{
				ServerName: "www.google.com",
			}
			tcpDialer := clientNet.NewDialerWithResolver(log.Log, clientNet.NewStdlibResolver(log.Log))
			tlsHandshaker := clientNet.NewTLSHandshakerStdlib(log.Log)
			tlsDialer := netxlite.NewTLSDialerWithConfig(tcpDialer, tlsHandshaker, tlsConfig)

			conn, err := tlsDialer.DialTLSContext(context.Background(), "tcp", proxy.Endpoint())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			tconn := conn.(netxlite.TLSConn)
			connstate := tconn.ConnectionState()
			t.Logf("%+v", connstate)
		})
	}
}
