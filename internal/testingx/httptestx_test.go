package testingx_test

// These tests are in a separate package because we need to import netxlite
// which otherwise creates a circular dependency with netxlite tests

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestHTTPTestxWithStdlib(t *testing.T) {
	// testcase describes a single testcase in this func
	type testcase struct {
		// name is the name of the test case
		name string

		// constructor constructs the HTTP server
		constructor func() *testingx.HTTPServer

		// timeout is the timeout to configure for the context
		timeout time.Duration

		// expectErr is the expected error
		expectErr error

		// expectCode is the expected status code
		expectCode int

		// expectBody is the expected response body
		expectBody []byte
	}

	// create server's CA
	serverCA := netem.MustNewCA()

	testcases := []testcase{
		/*
		 * HTTP
		 */
		{
			name: "with HTTP and the HTTPHandlerBlockpage451 handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServer(testingx.HTTPHandlerBlockpage451())
			},
			timeout:    10 * time.Second,
			expectErr:  nil,
			expectCode: http.StatusUnavailableForLegalReasons,
			expectBody: testingx.HTTPBlockpage451,
		}, {
			name: "with HTTP and the HTTPHandlerEOF handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServer(testingx.HTTPHandlerEOF())
			},
			timeout:    10 * time.Second,
			expectErr:  io.EOF,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTP and the HTTPHandlerReset handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
			},
			timeout:    10 * time.Second,
			expectErr:  netxlite.ECONNRESET,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTP and the HTTPHandlerTimeout handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServer(testingx.HTTPHandlerTimeout())
			},
			timeout:    1 * time.Second,
			expectErr:  context.DeadlineExceeded,
			expectCode: 0,
			expectBody: []byte{},
		},

		/*
		 * HTTPS
		 */
		{
			name: "with HTTPS and the HTTPHandlerBlockpage451 handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(
					testingx.HTTPHandlerBlockpage451(),
					serverCA,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  nil,
			expectCode: http.StatusUnavailableForLegalReasons,
			expectBody: testingx.HTTPBlockpage451,
		}, {
			name: "with HTTPS and the HTTPHandlerEOF handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(
					testingx.HTTPHandlerEOF(),
					serverCA,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  io.EOF,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTPS and the HTTPHandlerReset handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(
					testingx.HTTPHandlerReset(),
					serverCA,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  netxlite.ECONNRESET,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTPS and the HTTPHandlerTimeout handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(
					testingx.HTTPHandlerTimeout(),
					serverCA,
					"webserver.local",
				)
			},
			timeout:    1 * time.Second,
			expectErr:  context.DeadlineExceeded,
			expectCode: 0,
			expectBody: []byte{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// construct the HTTP server we're testing
			srvr := tc.constructor()
			defer srvr.Close()

			// create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// create the HTTP request
			req := runtimex.Try1(http.NewRequestWithContext(ctx, "GET", srvr.URL, nil))

			// create the HTTP client (we need to do more work than normally because
			// we MUST correctly set the TLS dialer configuration)
			netx := &netxlite.Netx{}
			tcpDialer := netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log))
			tlsHandshaker := netx.NewTLSHandshakerStdlib(log.Log)
			tlsDialer := netxlite.NewTLSDialerWithConfig(
				tcpDialer, tlsHandshaker, &tls.Config{RootCAs: srvr.X509CertPool})
			// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
			// function, but we can probably avoid using it, given that this code is
			// not using tracing and does not care about those quirks.
			txp := netxlite.NewHTTPTransport(log.Log, tcpDialer, tlsDialer)
			client := netxlite.NewHTTPClient(txp)

			// issue the request and get the response headers
			resp, err := client.Do(req)

			// handle error
			switch {
			case tc.expectErr == nil && err != nil:
				t.Fatal("expected", tc.expectErr, "but got", err)

			case tc.expectErr != nil && err == nil:
				t.Fatal("expected", tc.expectErr, "but got", err)

			case tc.expectErr != nil && err != nil:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("expected", tc.expectErr, "but got", err)
				}
				return

			default:
				// fallthrough
			}

			// make sure we'll close the response body
			defer resp.Body.Close()

			// make sure the status code is the expected one
			if resp.StatusCode != tc.expectCode {
				t.Fatal("unexpected status code", resp.StatusCode)
			}

			// read the response body until completion
			//
			// implementation note: a timeout here would cause us to hang forever but for now
			// this is fine because we're only timing out during the round trip
			rawBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			// compare to the expected body
			if diff := cmp.Diff(testingx.HTTPBlockpage451, rawBody); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestHTTPTestxWithNetem(t *testing.T) {
	// testcase describes a single testcase in this func
	type testcase struct {
		// name is the name of the test case
		name string

		// reasonToSkip is the reason why we should skip this test
		// or empty if there's no need to skip this test
		reasonToSkip string

		// constructor constructs the HTTP server
		constructor func(unet *netem.UNetStack) *testingx.HTTPServer

		// timeout is the timeout to configure for the context
		timeout time.Duration

		// expectErr is the expected error
		expectErr error

		// expectCode is the expected status code
		expectCode int

		// expectBody is the expected response body
		expectBody []byte
	}

	testcases := []testcase{
		/*
		 * HTTP
		 */
		{
			name: "with HTTP and the HTTPHandlerBlockpage451 handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 80,
					},
					unet,
					testingx.HTTPHandlerBlockpage451(),
				)
			},
			timeout:    10 * time.Second,
			expectErr:  nil,
			expectCode: http.StatusUnavailableForLegalReasons,
			expectBody: testingx.HTTPBlockpage451,
		}, {
			name: "with HTTP and the HTTPHandlerEOF handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 80,
					},
					unet,
					testingx.HTTPHandlerEOF(),
				)
			},
			timeout:    10 * time.Second,
			expectErr:  io.EOF,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name:         "with HTTP and the HTTPHandlerReset handler",
			reasonToSkip: "GVisor implements SO_LINGER but there is no gonet.TCPConn.SetLinger",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 80,
					},
					unet,
					testingx.HTTPHandlerReset(),
				)
			},
			timeout:    10 * time.Second,
			expectErr:  netxlite.ECONNRESET,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTP and the HTTPHandlerTimeout handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 80,
					},
					unet,
					testingx.HTTPHandlerTimeout(),
				)
			},
			timeout:    1 * time.Second,
			expectErr:  context.DeadlineExceeded,
			expectCode: 0,
			expectBody: []byte{},
		},

		/*
		 * HTTPS
		 */
		{
			name: "with HTTPS and the HTTPHandlerBlockpage451 handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLSEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 443,
					},
					unet,
					testingx.HTTPHandlerBlockpage451(),
					unet,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  nil,
			expectCode: http.StatusUnavailableForLegalReasons,
			expectBody: testingx.HTTPBlockpage451,
		}, {
			name: "with HTTPS and the HTTPHandlerEOF handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLSEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 443,
					},
					unet,
					testingx.HTTPHandlerEOF(),
					unet,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  io.EOF,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name:         "with HTTPS and the HTTPHandlerReset handler",
			reasonToSkip: "GVisor implements SO_LINGER but there is no gonet.TCPConn.SetLinger",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLSEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 443,
					},
					unet,
					testingx.HTTPHandlerReset(),
					unet,
					"webserver.local",
				)
			},
			timeout:    10 * time.Second,
			expectErr:  netxlite.ECONNRESET,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTPS and the HTTPHandlerTimeout handler",
			constructor: func(unet *netem.UNetStack) *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLSEx(
					&net.TCPAddr{
						IP:   net.ParseIP(unet.IPAddress()),
						Port: 443,
					},
					unet,
					testingx.HTTPHandlerTimeout(),
					unet,
					"webserver.local",
				)
			},
			timeout:    1 * time.Second,
			expectErr:  context.DeadlineExceeded,
			expectCode: 0,
			expectBody: []byte{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// make sure we should not skip this test
			if tc.reasonToSkip != "" {
				t.Skip(tc.reasonToSkip)
			}

			// create a star topology for hosting the test
			topology := netem.MustNewStarTopology(log.Log)
			defer topology.Close()

			// create a common link config
			linkConfig := &netem.LinkConfig{
				LeftToRightDelay: time.Millisecond,
				RightToLeftDelay: time.Millisecond,
			}

			// create the server stack
			serverStack := runtimex.Try1(topology.AddHost("10.0.0.1", "10.0.0.1", linkConfig))

			// create a DNS configuration
			dnsConfig := netem.NewDNSConfig()
			dnsConfig.AddRecord("dns.google", "", "10.0.0.1")

			// create a DNS server running on the server stack
			dnsServer := runtimex.Try1(netem.NewDNSServer(log.Log, serverStack, "10.0.0.1", dnsConfig))
			defer dnsServer.Close()

			// construct the HTTP server we're testing
			srvr := tc.constructor(serverStack)
			defer srvr.Close()

			// create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// construct the client stack
			clientStack := runtimex.Try1(topology.AddHost("10.0.0.2", "10.0.0.1", linkConfig))

			// use the client stack as netxlite's tproxy
			netxlite.WithCustomTProxy(&netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}, func() {

				// create the HTTP request
				req := runtimex.Try1(http.NewRequestWithContext(ctx, "GET", srvr.URL, nil))

				// create the HTTP client (we need to do more work than normally because
				// we MUST correctly set the TLS dialer configuration)
				netx := &netxlite.Netx{}
				tcpDialer := netx.NewDialerWithResolver(log.Log, netx.NewStdlibResolver(log.Log))
				tlsHandshaker := netx.NewTLSHandshakerStdlib(log.Log)
				tlsDialer := netxlite.NewTLSDialerWithConfig(
					tcpDialer, tlsHandshaker, &tls.Config{RootCAs: srvr.X509CertPool})
				// TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
				// function, but we can probably avoid using it, given that this code is
				// not using tracing and does not care about those quirks.
				txp := netxlite.NewHTTPTransport(log.Log, tcpDialer, tlsDialer)
				client := netxlite.NewHTTPClient(txp)

				// issue the request and get the response headers
				resp, err := client.Do(req)

				// handle error
				switch {
				case tc.expectErr == nil && err != nil:
					t.Fatal("expected", tc.expectErr, "but got", err)

				case tc.expectErr != nil && err == nil:
					t.Fatal("expected", tc.expectErr, "but got", err)

				case tc.expectErr != nil && err != nil:
					if !errors.Is(err, tc.expectErr) {
						t.Fatal("expected", tc.expectErr, "but got", err)
					}
					return

				default:
					// fallthrough
				}

				// make sure we'll close the response body
				defer resp.Body.Close()

				// make sure the status code is the expected one
				if resp.StatusCode != tc.expectCode {
					t.Fatal("unexpected status code", resp.StatusCode)
				}

				// read the response body until completion
				//
				// implementation note: a timeout here would cause us to hang forever but for now
				// this is fine because we're only timing out during the round trip
				rawBody, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}

				// compare to the expected body
				if diff := cmp.Diff(testingx.HTTPBlockpage451, rawBody); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}

func TestHTTPHandlerResetWhileReadingBody(t *testing.T) {
	// create a server for testing the given handler
	server := testingx.MustNewHTTPServer(testingx.HTTPHandlerResetWhileReadingBody())
	defer server.Close()

	// create a suitable HTTP transport using netxlite
	netx := &netxlite.Netx{Underlying: nil}
	dialer := netx.NewDialerWithoutResolver(log.Log)
	handshaker := netx.NewTLSHandshakerStdlib(log.Log)
	tlsDialer := netxlite.NewTLSDialer(dialer, handshaker)
	txp := netxlite.NewHTTPTransportWithOptions(log.Log, dialer, tlsDialer)

	// create the request
	req := runtimex.Try1(http.NewRequest("GET", server.URL, nil))

	// perform the round trip
	resp, err := txp.RoundTrip(req)

	// we do not expect an error during the round trip
	if err != nil {
		t.Fatal(err)
	}

	// make sure we close the body
	defer resp.Body.Close()

	// start reading the response where we expect to see a RST
	respbody, err := netxlite.ReadAllContext(req.Context(), resp.Body)

	// verify we received a connection reset
	if !errors.Is(err, netxlite.ECONNRESET) {
		t.Fatal("expected ECONNRESET, got", err)
	}

	// make sure we've got no bytes
	if len(respbody) != 0 {
		t.Fatal("expected to see zero bytes here")
	}
}
