package testingx_test

// These tests are in a separate package because we need to import netxlite
// which otherwise creates a circular dependency with netxlite tests

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestTLSHandlerWithStdlib(t *testing.T) {
	// testcase is a test case implemented by this func
	type testcase struct {
		// name is the name of the test case
		name string

		// newHandler is the factory for creating a new handler
		newHandler func() testingx.TLSHandler

		// timeout is the TLS handshake timeout
		timeout time.Duration

		// expectErr is the expected TLS handshake error
		expectErr error

		// expectBody is the text we expect to receive otherwise
		expectBody []byte
	}

	// create server's CA and leaf certificate
	serverCA := netem.MustNewCA()
	serverCert := serverCA.MustNewTLSCertificate("www.example.com")

	testcases := []testcase{{
		name: "with TLSHandlerTimeout",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerTimeout()
		},
		timeout:    1 * time.Second,
		expectErr:  errors.New(netxlite.FailureGenericTimeoutError),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerSendAlert",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerSendAlert(testingx.TLSAlertUnrecognizedName)
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureSSLInvalidHostname),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerEOF",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerEOF()
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureEOFError),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerReset",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerReset()
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureConnectionReset),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerHandshakeAndWriteText",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerHandshakeAndWriteText(serverCert, testingx.HTTPBlockpage451)
		},
		timeout:    10 * time.Second,
		expectErr:  nil,
		expectBody: testingx.HTTPBlockpage451,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// create the server running in the background
			server := testingx.MustNewTLSServer(tc.newHandler())
			defer server.Close()

			// create TLS config with a specific SNI
			tlsConfig := &tls.Config{
				RootCAs:    serverCA.DefaultCertPool(),
				ServerName: "www.example.com",
			}

			// create a TLS dialer
			tcpDialer := netxlite.NewDialerWithoutResolver(log.Log)
			tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
			tlsDialer := netxlite.NewTLSDialerWithConfig(tcpDialer, tlsHandshaker, tlsConfig)

			// create a context with a timeout
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			// establish a TLS connection
			tlsConn, err := tlsDialer.DialTLSContext(ctx, "tcp", server.Endpoint())

			// check the result of the handshake
			switch {
			case tc.expectErr == nil && err != nil:
				t.Fatal("expected", tc.expectErr, "but got", err)

			case tc.expectErr != nil && err == nil:
				t.Fatal("expected", tc.expectErr, "but got", err)

			case tc.expectErr != nil && err != nil:
				if err.Error() != tc.expectErr.Error() {
					t.Fatal("expected", tc.expectErr, "but got", err)
				}
				return

			default:
				// fallthrough
			}

			// make sure we close the connection
			defer tlsConn.Close()

			// read bytes from the connection
			data, err := io.ReadAll(io.LimitReader(tlsConn, 1<<14))
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expectBody, data); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTLSHandlerWithNetem(t *testing.T) {
	// testcase is a test case implemented by this func
	type testcase struct {
		// name is the name of the test case
		name string

		// reasonToSkip indicates the reason why we should skip this test
		reasonToSkip string

		// newHandler is the factory for creating a new handler
		newHandler func() testingx.TLSHandler

		// timeout is the TLS handshake timeout
		timeout time.Duration

		// expectErr is the expected TLS handshake error
		expectErr error

		// expectBody is the text we expect to receive otherwise
		expectBody []byte
	}

	// create server's CA and leaf certificate
	serverCA := netem.MustNewCA()
	serverCert := serverCA.MustNewTLSCertificate("www.example.com")

	testcases := []testcase{{
		name: "with TLSHandlerTimeout",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerTimeout()
		},
		timeout:    1 * time.Second,
		expectErr:  errors.New(netxlite.FailureGenericTimeoutError),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerSendAlert",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerSendAlert(testingx.TLSAlertUnrecognizedName)
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureSSLInvalidHostname),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerEOF",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerEOF()
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureEOFError),
		expectBody: []byte{},
	}, {
		name:         "with TLSHandlerReset",
		reasonToSkip: "GVisor implements SO_LINGER but there is no gonet.TCPConn.SetLinger",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerReset()
		},
		timeout:    10 * time.Second,
		expectErr:  errors.New(netxlite.FailureConnectionReset),
		expectBody: []byte{},
	}, {
		name: "with TLSHandlerHandshakeAndWriteText",
		newHandler: func() testingx.TLSHandler {
			return testingx.TLSHandlerHandshakeAndWriteText(serverCert, testingx.HTTPBlockpage451)
		},
		timeout:    10 * time.Second,
		expectErr:  nil,
		expectBody: testingx.HTTPBlockpage451,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.reasonToSkip != "" {
				t.Skip(tc.reasonToSkip)
			}

			// create a star topology for this test case
			topology := netem.MustNewStarTopology(log.Log)
			defer topology.Close()

			// create the server
			serverStack := runtimex.Try1(topology.AddHost("142.251.209.14", "0.0.0.0", &netem.LinkConfig{}))
			server := testingx.MustNewTLSServerEx(
				&net.TCPAddr{IP: net.IPv4(142, 251, 209, 14), Port: 443},
				serverStack,
				tc.newHandler(),
			)
			defer server.Close()

			// create the client stack
			clientStack := runtimex.Try1(topology.AddHost("10.0.0.2", "142.251.209.14", &netem.LinkConfig{}))

			// use the client stack
			netxlite.WithCustomTProxy(&netxlite.NetemUnderlyingNetworkAdapter{UNet: clientStack}, func() {
				// create TLS config with a specific SNI
				tlsConfig := &tls.Config{
					RootCAs:    serverCA.DefaultCertPool(),
					ServerName: "www.example.com",
				}

				// create a TLS dialer
				tcpDialer := netxlite.NewDialerWithoutResolver(log.Log)
				tlsHandshaker := netxlite.NewTLSHandshakerStdlib(log.Log)
				tlsDialer := netxlite.NewTLSDialerWithConfig(tcpDialer, tlsHandshaker, tlsConfig)

				// create a context with a timeout
				ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
				defer cancel()

				// establish a TLS connection
				tlsConn, err := tlsDialer.DialTLSContext(ctx, "tcp", server.Endpoint())

				// check the result of the handshake
				switch {
				case tc.expectErr == nil && err != nil:
					t.Fatal("expected", tc.expectErr, "but got", err)

				case tc.expectErr != nil && err == nil:
					t.Fatal("expected", tc.expectErr, "but got", err)

				case tc.expectErr != nil && err != nil:
					if err.Error() != tc.expectErr.Error() {
						t.Fatal("expected", tc.expectErr, "but got", err)
					}
					return

				default:
					// fallthrough
				}

				// make sure we close the connection
				defer tlsConn.Close()

				// read bytes from the connection
				data, err := io.ReadAll(io.LimitReader(tlsConn, 1<<14))
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(tc.expectBody, data); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}
