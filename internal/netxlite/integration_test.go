package netxlite_test

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingquic"
	"github.com/ooni/probe-cli/v3/internal/testingx"
	"github.com/quic-go/quic-go"
	utls "gitlab.com/yawning/utls.git"
)

// This set of integration tests ensures that we continue to
// be able to measure the conditions we care about

func TestMeasureWithSystemResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	//
	// Measurement conditions we care about:
	//
	// - success
	//
	// - nxdomain
	//
	// - timeout
	//

	t.Run("on success", func(t *testing.T) {
		r := netxlite.NewStdlibResolver(log.Log)
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "dns.google.com")
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("expected non-nil result here")
		}
	})

	t.Run("for nxdomain", func(t *testing.T) {
		r := netxlite.NewStdlibResolver(log.Log)
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "www.ooni.nonexistent")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil result here")
		}
	})

	t.Run("for timeout", func(t *testing.T) {
		r := netxlite.NewStdlibResolver(log.Log)
		defer r.CloseIdleConnections()
		const timeout = time.Nanosecond
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// Implementation note: Windows' resolver has caching so back to back tests
		// will fail unless we query for something that could bypass the cache itself
		// e.g. a domain containing a few random letters
		addrs, err := r.LookupHost(ctx, randx.Letters(7)+".ooni.nonexistent")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			if runtime.GOOS == "windows" {
				t.Skip("https://github.com/ooni/probe/issues/2535")
			}
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil result here")
		}
	})
}

func TestMeasureWithUDPResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	//
	// Measurement conditions we care about:
	//
	// - success
	//
	// - nxdomain
	//
	// - refused
	//
	// - timeout
	//

	t.Run("on success", func(t *testing.T) {
		netx := &netxlite.Netx{}
		dlr := netx.NewDialerWithoutResolver(log.Log)
		r := netxlite.NewParallelUDPResolver(log.Log, dlr, "8.8.4.4:53")
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "dns.google.com")
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("expected non-nil result here")
		}
	})

	t.Run("for nxdomain", func(t *testing.T) {
		udpAddr := &net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 0,
		}
		dnsRtx := testingx.NewDNSRoundTripperNXDOMAIN()
		listener := testingx.MustNewDNSOverUDPListener(udpAddr, &testingx.DNSOverUDPListenerStdlib{}, dnsRtx)
		defer listener.Close()
		netx := &netxlite.Netx{}
		dlr := netx.NewDialerWithoutResolver(log.Log)
		r := netxlite.NewParallelUDPResolver(log.Log, dlr, listener.LocalAddr().String())
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "ooni.org")
		if err == nil || err.Error() != netxlite.FailureDNSNXDOMAINError {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil result here")
		}
	})

	t.Run("for refused", func(t *testing.T) {
		udpAddr := &net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 0,
		}
		dnsRtx := testingx.NewDNSRoundTripperRefused()
		listener := testingx.MustNewDNSOverUDPListener(udpAddr, &testingx.DNSOverUDPListenerStdlib{}, dnsRtx)
		defer listener.Close()
		netx := &netxlite.Netx{}
		dlr := netx.NewDialerWithoutResolver(log.Log)
		r := netxlite.NewParallelUDPResolver(log.Log, dlr, listener.LocalAddr().String())
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "ooni.org")
		if err == nil || err.Error() != netxlite.FailureDNSRefusedError {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil result here")
		}
	})

	t.Run("for timeout", func(t *testing.T) {
		udpAddr := &net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 0,
		}
		dnsRtx := testingx.NewDNSRoundTripperSimulateTimeout(time.Millisecond, errors.New("mocked error"))
		listener := testingx.MustNewDNSOverUDPListener(udpAddr, &testingx.DNSOverUDPListenerStdlib{}, dnsRtx)
		defer listener.Close()
		netx := &netxlite.Netx{}
		dlr := netx.NewDialerWithoutResolver(log.Log)
		r := netxlite.NewParallelUDPResolver(log.Log, dlr, listener.LocalAddr().String())
		defer r.CloseIdleConnections()
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "ooni.org")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil result here")
		}
	})
}

func TestMeasureWithDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	//
	// Measurement conditions we care about:
	//
	// - success
	//
	// - connection refused
	//
	// - timeout
	//

	t.Run("on success", func(t *testing.T) {
		netx := &netxlite.Netx{}
		d := netx.NewDialerWithoutResolver(log.Log)
		defer d.CloseIdleConnections()
		ctx := context.Background()
		conn, err := d.DialContext(ctx, "tcp", "8.8.4.4:443")
		if err != nil {
			t.Fatal(err)
		}
		if conn == nil {
			t.Fatal("expected non-nil conn here")
		}
		conn.Close()
	})

	t.Run("on connection refused", func(t *testing.T) {
		netx := &netxlite.Netx{}
		d := netx.NewDialerWithoutResolver(log.Log)
		defer d.CloseIdleConnections()
		ctx := context.Background()
		// Here we assume that no-one is listening on 127.0.0.1:1
		conn, err := d.DialContext(ctx, "tcp", "127.0.0.1:1")
		if err == nil || err.Error() != netxlite.FailureConnectionRefused {
			t.Fatal("not the error we expected", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
	})

	t.Run("on timeout", func(t *testing.T) {
		// Note: this test was flaky sometimes on macOS. I've seen in
		// particular this failure on 2021-09-29:
		//
		// ```
		// --- FAIL: TestMeasureWithDialer (8.25s)
		// --- FAIL: TestMeasureWithDialer/on_timeout (8.22s)
		//   integration_test.go:233: not the error we expected timed_out
		// ```
		//
		// My explanation of this failure is that the ETIMEDOUT from
		// the kernel races with the timeout we've configured. For this
		// reason, I have set a smaller context timeout (see below).
		//
		netx := &netxlite.Netx{}
		d := netx.NewDialerWithoutResolver(log.Log)
		defer d.CloseIdleConnections()
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		// Here we assume 8.8.4.4:1 is filtered
		conn, err := d.DialContext(ctx, "tcp", "8.8.4.4:1")
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("not the error we expected", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
	})
}

func TestMeasureWithTLSHandshaker(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	//
	// Measurement conditions we care about:
	//
	// - success
	//
	// - connection reset
	//
	// - timeout
	//
	// - tls unrecognized name alert
	//

	dial := func(ctx context.Context, address string) (net.Conn, error) {
		netx := &netxlite.Netx{}
		d := netx.NewDialerWithoutResolver(log.Log)
		return d.DialContext(ctx, "tcp", address)
	}

	successFlow := func(th model.TLSHandshaker) error {
		ctx := context.Background()
		conn, err := dial(ctx, "8.8.4.4:443")
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer conn.Close()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: "dns.google",
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    nil,
		}
		tconn, err := th.Handshake(ctx, conn, config)
		if err != nil {
			return fmt.Errorf("tls handshake failed: %w", err)
		}
		tconn.Close()
		return nil
	}

	connectionResetFlow := func(th model.TLSHandshaker) error {
		server := testingx.MustNewTLSServer(testingx.TLSHandlerReset())
		defer server.Close()
		ctx := context.Background()
		conn, err := dial(ctx, server.Endpoint())
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer conn.Close()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: "dns.google",
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    nil,
		}
		tconn, err := th.Handshake(ctx, conn, config)
		if err == nil {
			return fmt.Errorf("tls handshake succeded unexpectedly")
		}
		if err.Error() != netxlite.FailureConnectionReset {
			return fmt.Errorf("not the error we expected: %w", err)
		}
		if tconn != nil {
			return fmt.Errorf("expected nil tconn here")
		}
		return nil
	}

	timeoutFlow := func(th model.TLSHandshaker) error {
		server := testingx.MustNewTLSServer(testingx.TLSHandlerTimeout())
		defer server.Close()
		ctx := context.Background()
		conn, err := dial(ctx, server.Endpoint())
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer conn.Close()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: "dns.google",
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    nil,
		}
		tconn, err := th.Handshake(ctx, conn, config)
		if err == nil {
			return fmt.Errorf("tls handshake succeded unexpectedly")
		}
		if err.Error() != netxlite.FailureGenericTimeoutError {
			return fmt.Errorf("not the error we expected: %w", err)
		}
		if tconn != nil {
			return fmt.Errorf("expected nil tconn here")
		}
		return nil
	}

	tlsUnrecognizedNameFlow := func(th model.TLSHandshaker) error {
		server := testingx.MustNewTLSServer(testingx.TLSHandlerSendAlert(testingx.TLSAlertUnrecognizedName))
		defer server.Close()
		ctx := context.Background()
		conn, err := dial(ctx, server.Endpoint())
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer conn.Close()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: "dns.google",
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    nil,
		}
		tconn, err := th.Handshake(ctx, conn, config)
		if err == nil {
			return fmt.Errorf("tls handshake succeded unexpectedly")
		}
		if err.Error() != netxlite.FailureSSLInvalidHostname {
			return fmt.Errorf("not the error we expected: %w", err)
		}
		if tconn != nil {
			return fmt.Errorf("expected nil tconn here")
		}
		return nil
	}

	t.Run("for stdlib handshaker", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerStdlib(log.Log)
			err := successFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on connection reset", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerStdlib(log.Log)
			err := connectionResetFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on timeout", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerStdlib(log.Log)
			err := timeoutFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on TLS unrecognized name alert", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerStdlib(log.Log)
			err := tlsUnrecognizedNameFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("for utls handshaker", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
			err := successFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on connection reset", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
			err := connectionResetFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on timeout", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
			err := timeoutFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("on TLS unrecognized name alert", func(t *testing.T) {
			th := netxlite.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
			err := tlsUnrecognizedNameFlow(th)
			if err != nil {
				t.Fatal(err)
			}
		})
	})
}

func TestMeasureWithQUICDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	// TODO(bassosimone): here we're not testing the case in which
	// the certificate is invalid for the required SNI.

	//
	// Measurement conditions we care about:
	//
	// - success
	//
	// - timeout
	//

	t.Run("on success", func(t *testing.T) {
		netx := &netxlite.Netx{}
		ql := netxlite.NewUDPListener()
		d := netx.NewQUICDialerWithoutResolver(ql, log.Log)
		defer d.CloseIdleConnections()
		ctx := context.Background()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: testingquic.MustDomain(),
			NextProtos: []string{"h3"},
			RootCAs:    nil,
		}
		sess, err := d.DialContext(ctx, testingquic.MustEndpoint("443"), config, &quic.Config{})
		if err != nil {
			t.Fatal(err)
		}
		if sess == nil {
			t.Fatal("expected non-nil sess here")
		}
		sess.CloseWithError(0, "")
	})

	t.Run("on timeout", func(t *testing.T) {
		netx := &netxlite.Netx{}
		ql := netxlite.NewUDPListener()
		d := netx.NewQUICDialerWithoutResolver(ql, log.Log)
		defer d.CloseIdleConnections()
		ctx := context.Background()
		// See https://github.com/ooni/probe/issues/2413 to understand
		// why we're using nil to force netxlite to use the cached
		// default Mozilla cert pool.
		config := &tls.Config{
			ServerName: testingquic.MustDomain(),
			NextProtos: []string{"h3"},
			RootCAs:    nil,
		}
		// Here we assume <target-address>:1 is filtered
		sess, err := d.DialContext(ctx, testingquic.MustEndpoint("1"), config, &quic.Config{})
		if err == nil || err.Error() != netxlite.FailureGenericTimeoutError {
			t.Fatal("not the error we expected", err)
		}
		if sess != nil {
			t.Fatal("expected nil sess here")
		}
	})
}

func TestHTTPTransport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		netx := &netxlite.Netx{}
		d := netx.NewDialerWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
		td := netxlite.NewTLSDialer(d, netxlite.NewTLSHandshakerStdlib(log.Log))
		txp := netxlite.NewHTTPTransport(log.Log, d, td)
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://www.google.com/robots.txt")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		client.CloseIdleConnections()
	})

	t.Run("we can read the body when the connection is closed", func(t *testing.T) {
		// See https://github.com/ooni/probe/issues/1965
		srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hj := w.(http.Hijacker) // panic if not possible
			conn, bufrw, err := hj.Hijack()
			runtimex.PanicOnError(err, "hj.Hijack failed")
			bufrw.WriteString("HTTP/1.0 302 Found\r\n")
			bufrw.WriteString("Location: /text\r\n\r\n")
			bufrw.Flush()
			conn.Close()
		}))
		defer srvr.Close()
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPTransportStdlib has QUIRKS but we
		// don't actually care about those QUIRKS in this context
		netx := &netxlite.Netx{}
		txp := netx.NewHTTPTransportStdlib(model.DiscardLogger)
		req, err := http.NewRequest("GET", srvr.URL, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := txp.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := netxlite.ReadAllContext(req.Context(), resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(data))
	})
}

func TestHTTP3Transport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	t.Run("works as intended", func(t *testing.T) {
		netx := &netxlite.Netx{}
		d := netx.NewQUICDialerWithResolver(
			netxlite.NewUDPListener(),
			log.Log,
			netxlite.NewStdlibResolver(log.Log),
		)
		txp := netxlite.NewHTTP3Transport(log.Log, d, &tls.Config{})
		client := &http.Client{Transport: txp}
		URL := (&url.URL{Scheme: "https", Host: testingquic.MustDomain(), Path: "/"}).String()
		resp, err := client.Get(URL)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		txp.CloseIdleConnections()
	})
}
