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
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
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
				return testingx.MustNewHTTPServerTLS(testingx.HTTPHandlerBlockpage451())
			},
			timeout:    10 * time.Second,
			expectErr:  nil,
			expectCode: http.StatusUnavailableForLegalReasons,
			expectBody: testingx.HTTPBlockpage451,
		}, {
			name: "with HTTPS and the HTTPHandlerEOF handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(testingx.HTTPHandlerEOF())
			},
			timeout:    10 * time.Second,
			expectErr:  io.EOF,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTPS and the HTTPHandlerReset handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(testingx.HTTPHandlerReset())
			},
			timeout:    10 * time.Second,
			expectErr:  netxlite.ECONNRESET,
			expectCode: 0,
			expectBody: []byte{},
		}, {
			name: "with HTTPS and the HTTPHandlerTimeout handler",
			constructor: func() *testingx.HTTPServer {
				return testingx.MustNewHTTPServerTLS(testingx.HTTPHandlerTimeout())
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
			tcpDialer := netxlite.NewDialerWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
			tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
			tlsDialer := netxlite.NewTLSDialerWithConfig(
				tcpDialer, tlsHandshaker, &tls.Config{RootCAs: srvr.X509CertPool})
			txp := netxlite.NewHTTPTransportLegacy(log.Log, tcpDialer, tlsDialer)
			client := netxlite.NewHTTPClientLegacy(txp)

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
			topology := runtimex.Try1(netem.NewStarTopology(log.Log))
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
				tcpDialer := netxlite.NewDialerWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
				tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
				tlsDialer := netxlite.NewTLSDialerWithConfig(
					tcpDialer, tlsHandshaker, &tls.Config{RootCAs: srvr.X509CertPool})
				txp := netxlite.NewHTTPTransportLegacy(log.Log, tcpDialer, tlsDialer)
				client := netxlite.NewHTTPClientLegacy(txp)

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

func TestHTTPHandlerProxy(t *testing.T) {
	expectedBody := []byte("Google is built by a large team of engineers, designers, researchers, robots, and others in many different sites across the globe. It is updated continuously, and built with more tools and technologies than we can shake a stick at. If you'd like to help us out, see careers.google.com.\n")

	type testcase struct {
		name      string
		construct func() (*netxlite.Netx, string, []io.Closer)
		short     bool
	}

	testcases := []testcase{
		{
			name: "using the real network",
			construct: func() (*netxlite.Netx, string, []io.Closer) {
				var closers []io.Closer

				netx := &netxlite.Netx{
					Underlying: nil, // so we're using the real network
				}

				proxyServer := testingx.MustNewHTTPServer(testingx.HTTPHandlerProxy(log.Log, netx))
				closers = append(closers, proxyServer)

				return netx, proxyServer.URL, closers
			},
			short: false,
		},

		{
			name: "using netem",
			construct: func() (*netxlite.Netx, string, []io.Closer) {
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

				wwwServer := testingx.MustNewHTTPServerEx(
					&net.TCPAddr{IP: net.IPv4(142, 251, 209, 14), Port: 80},
					wwwStack,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Write(expectedBody)
					}),
				)
				closers = append(closers, wwwServer)

				proxyServer := testingx.MustNewHTTPServerEx(
					&net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 80},
					proxyStack,
					testingx.HTTPHandlerProxy(log.Log, &netxlite.Netx{
						Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: proxyStack},
					}),
				)
				closers = append(closers, proxyServer)

				clientNet := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}}
				return clientNet, proxyServer.URL, closers
			},
			short: true,
		}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.short && testing.Short() {
				t.Skip("skip test in short mode")
			}

			netx, proxyURL, closers := tc.construct()
			defer func() {
				for _, closer := range closers {
					closer.Close()
				}
			}()

			URL := runtimex.Try1(url.Parse(proxyURL))
			URL.Path = "/humans.txt"

			req := runtimex.Try1(http.NewRequest("GET", URL.String(), nil))
			req.Host = "www.google.com"

			//log.SetLevel(log.DebugLevel)

			txp := netx.NewHTTPTransportStdlibLegacy(log.Log)
			client := netxlite.NewHTTPClientLegacy(txp)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				t.Fatal("expected to see 200, got", resp.StatusCode)
			}

			t.Logf("%+v", resp)

			body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			t.Logf("%s", string(body))

			if diff := cmp.Diff(expectedBody, body); diff != "" {
				t.Fatal(diff)
			}
		})
	}

	t.Run("rejects requests without a host header", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.HTTPHandlerProxy(log.Log, netx)
		req := &http.Request{
			Host: "", // explicitly empty
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("rejects requests with a via header", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.HTTPHandlerProxy(log.Log, netx)
		req := &http.Request{
			Host: "www.example.com",
			Header: http.Header{
				"Via": {"antani/0.1.0"},
			},
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("rejects requests with a POST method", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			// all nil: panic if we hit the network
		}}
		handler := testingx.HTTPHandlerProxy(log.Log, netx)
		req := &http.Request{
			Host:   "www.example.com",
			Header: http.Header{},
			Method: http.MethodPost,
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})

	t.Run("returns 502 when the round trip fails", func(t *testing.T) {
		rr := httptest.NewRecorder()
		netx := &netxlite.Netx{Underlying: &mocks.UnderlyingNetwork{
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return nil, "", errors.New("mocked error")
			},
			MockGetaddrinfoResolverNetwork: func() string {
				return "antani"
			},
		}}
		handler := testingx.HTTPHandlerProxy(log.Log, netx)
		req := &http.Request{
			Host:   "www.example.com",
			Header: http.Header{},
			Method: http.MethodGet,
			URL:    &url.URL{},
		}
		handler.ServeHTTP(rr, req)
		res := rr.Result()
		if res.StatusCode != http.StatusBadGateway {
			t.Fatal("unexpected status code", res.StatusCode)
		}
	})
}
