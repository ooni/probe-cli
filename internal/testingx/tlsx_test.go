package testingx_test

// This test is in a separate package because we need to import netxlite
// which otherwise creates a circular dependency with netxlite tests

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
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

	// create MITM config
	mitm := testingx.MustNewTLSMITMProviderNetem()

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
			return testingx.TLSHandlerHandshakeAndWriteText(mitm, testingx.HTTPBlockpage451)
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
				RootCAs:    runtimex.Try1(mitm.DefaultCertPool()),
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
	t.Skip("test not implemented")
}
