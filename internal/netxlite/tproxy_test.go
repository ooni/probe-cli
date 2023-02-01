package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDefaultTProxy(t *testing.T) {
	t.Run("DialContext honours the timeout", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			// This test is here to give us confidence we're doing the right thing
			// in terms of the underlying Go API. It's not here to make sure the
			// github CI behaves exactly equally on Windows, Linux, macOS on edge
			// cases. So, it seems fine to just skip this test on Windows.
			//
			// TODO(https://github.com/ooni/probe/issues/2368).
			t.Skip("skip test on windows")
		}
		tp := &DefaultTProxy{}
		ctx := context.Background()
		conn, err := tp.DialContext(ctx, time.Nanosecond, "tcp", "1.1.1.1:443")
		if err == nil || !strings.HasSuffix(err.Error(), "i/o timeout") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}

func TestWithCustomTProxy(t *testing.T) {

	t.Run("we can override the default cert pool", func(t *testing.T) {
		srvr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(444)
		}))
		defer srvr.Close()

		// TODO(bassosimone): we need a more compact and ergonomic
		// way of overriding the underlying network
		tproxy := &mocks.UnderlyingNetwork{
			MockDialContext: func(ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
				return (&DefaultTProxy{}).DialContext(ctx, timeout, network, address)
			},
			MockListenUDP: func(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return (&DefaultTProxy{}).ListenUDP(network, addr)
			},
			MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
				return (&DefaultTProxy{}).GetaddrinfoLookupANY(ctx, domain)
			},
			MockGetaddrinfoResolverNetwork: func() string {
				return (&DefaultTProxy{}).GetaddrinfoResolverNetwork()
			},
			MockMaybeModifyPool: func(*x509.CertPool) *x509.CertPool {
				pool := x509.NewCertPool()
				pool.AddCert(srvr.Certificate())
				return pool
			},
		}

		WithCustomTProxy(tproxy, func() {
			clnt := NewHTTPClientStdlib(model.DiscardLogger)
			req, err := http.NewRequestWithContext(context.Background(), "GET", srvr.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := clnt.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 444 {
				t.Fatal("unexpected status code")
			}
		})
	})
}
