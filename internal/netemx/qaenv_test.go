package netemx_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TestQAEnv ensures that we can use a [netemx.QAEnv] to hijack [netxlite] function calls.
func TestQAEnv(t *testing.T) {

	// Here we're testing that:
	//
	// 1. we can get the expected private answer for www.example.com, meaning that
	// we are using the userspace TCP/IP stack defined by [Environment].
	t.Run("we can hijack getaddrinfo lookups", func(t *testing.T) {
		// create QA env
		env := netemx.NewQAEnv()
		defer env.Close()

		// configure DNS
		env.AddRecordToAllResolvers(
			"www.example.com",
			"netem.example.com", // CNAME
			"10.0.17.1",
			"10.0.17.2",
			"10.0.17.3",
		)

		env.Do(func() {
			// create stdlib resolver, which will use the underlying client stack
			// GetaddrinfoLookupANY method for the DNS lookup
			reso := netxlite.NewStdlibResolver(model.DiscardLogger)

			// lookup the hostname
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "www.example.com")

			// verify that the result is okay
			if err != nil {
				t.Fatal(err)
			}
			expectAddrs := []string{
				"10.0.17.1",
				"10.0.17.2",
				"10.0.17.3",
			}
			if diff := cmp.Diff(expectAddrs, addrs); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// Here we're testing that:
	//
	// 1. we can get the expected answer for quad8.com;
	//
	// 2. we connect to the expected address;
	//
	// 3. we can successfully TLS handshake for quad8.com;
	//
	// 4. we obtain the expected webpage.
	//
	// If all of this works, it means we're using the userspace TCP/IP
	// stack exported by the [Environment] struct.
	t.Run("we can hijack HTTPS requests", func(t *testing.T) {
		// create QA env
		env := netemx.NewQAEnv(
			netemx.QAEnvOptionHTTPServer("8.8.8.8", netemx.QAEnvDefaultHTTPHandler()),
		)
		defer env.Close()

		// configure DNS
		env.AddRecordToAllResolvers(
			"quad8.com",
			"", // CNAME
			"8.8.8.8",
		)

		env.Do(func() {
			// create client, which will use the underlying client stack's
			// DialContext method to dial connections
			client := netxlite.NewHTTPClientStdlib(model.DiscardLogger)

			// create request using a domain that has been configured in the
			// [Environment] we're using as valid. Note that we're using https
			// and this will work because the client stack also controls the
			// default CA pool through the DefaultCertPool method.
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// make sure the status code and the body match
			if resp.StatusCode != 200 {
				t.Fatal("expected to see 200, got", resp.StatusCode)
			}
			expectBody := []byte(netemx.QAEnvDefaultWebPage)
			gotBody, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expectBody, gotBody); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// Here we're testing that:
	//
	// 1. we can get the expected answer for quad8.com;
	//
	// 2. we can successfully QUIC handshake for quad8.com;
	//
	// 3. we obtain the expected webpage.
	//
	// If all of this works, it means we're using the userspace TCP/IP
	// stack exported by the [Environment] struct.
	t.Run("we can hijack HTTP3 requests", func(t *testing.T) {
		// create QA env
		env := netemx.NewQAEnv(
			netemx.QAEnvOptionHTTPServer("8.8.8.8", netemx.QAEnvDefaultHTTPHandler()),
		)
		defer env.Close()

		// configure DNS
		env.AddRecordToAllResolvers(
			"quad8.com",
			"", // CNAME
			"8.8.8.8",
		)

		env.Do(func() {
			// create an HTTP3 client
			txp := netxlite.NewHTTP3TransportStdlib(model.DiscardLogger)
			client := &http.Client{Transport: txp}

			// create the request; see above remarks for the HTTPS case
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// check the response and the body
			if resp.StatusCode != 200 {
				t.Fatal("expected to see 200, got", resp.StatusCode)
			}
			expectBody := []byte(netemx.QAEnvDefaultWebPage)
			gotBody, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expectBody, gotBody); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// This is like the one where we test for HTTPS. The idea here is to
	// be sure that we can set DPI rules affecting the client stack.
	t.Run("we can configure DPI rules", func(t *testing.T) {
		// create QA env
		env := netemx.NewQAEnv(
			netemx.QAEnvOptionHTTPServer("8.8.8.8", netemx.QAEnvDefaultHTTPHandler()),
		)
		defer env.Close()

		// configure DNS
		env.AddRecordToAllResolvers(
			"quad8.com",
			"", // CNAME
			"8.8.8.8",
		)

		// create DPI rule blocking the quad8.com SNI with RST
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "quad8.com",
		})

		env.Do(func() {
			// create client, which will use the underlying client stack's
			// DialContext method to dial connections
			client := netxlite.NewHTTPClientStdlib(model.DiscardLogger)

			// create the request
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)

			// make sure we got a connection RST by peer error
			if err == nil || err.Error() != netxlite.FailureConnectionReset {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
		})
	})
}
