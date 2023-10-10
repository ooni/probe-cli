package netxlite

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestTproxyNilSafeProvider(t *testing.T) {
	type testingstruct struct {
		provider *MaybeCustomUnderlyingNetwork
	}

	t.Run("when the pointer is nil", func(t *testing.T) {
		tsp := &testingstruct{}
		if tsp.provider.Get() != tproxySingleton() {
			t.Fatal("unexpected result")
		}
	})

	t.Run("when underlying is nil", func(t *testing.T) {
		tsp := &testingstruct{
			provider: &MaybeCustomUnderlyingNetwork{
				underlying: nil,
			},
		}
		if tsp.provider.Get() != tproxySingleton() {
			t.Fatal("unexpected result")
		}
	})

	t.Run("when underlying is set", func(t *testing.T) {
		expected := &mocks.UnderlyingNetwork{}
		tsp := &testingstruct{
			provider: &MaybeCustomUnderlyingNetwork{
				underlying: expected,
			},
		}
		if tsp.provider.Get() != expected {
			t.Fatal("unexpected result")
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
			MockDefaultCertPool: func() *x509.CertPool {
				pool := x509.NewCertPool()
				pool.AddCert(srvr.Certificate())
				return pool
			},
			MockDialTimeout: func() time.Duration {
				return defaultDialTimeout
			},
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return (&DefaultTProxy{}).DialContext(ctx, network, address)
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
		}

		WithCustomTProxy(tproxy, func() {
			// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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

// We generally do not listen here as part of other tests, since the listening
// functionality is mainly only use for testingx. So, here's a specific test for that.
func TestTproxyListenTCP(t *testing.T) {
	tproxy := &DefaultTProxy{}

	listener := runtimex.Try1(tproxy.ListenTCP("tcp", &net.TCPAddr{}))
	serverEndpoint := listener.Addr().String()

	// listen in a background goroutine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		conn := runtimex.Try1(listener.Accept())
		conn.Close()
		wg.Done()
	}()

	// dial in a background goroutine
	wg.Add(1)
	go func() {
		ctx := context.Background()
		conn := runtimex.Try1(tproxy.DialContext(ctx, "tcp", serverEndpoint))
		conn.Close()
		wg.Done()
	}()

	// wait for the goroutines to finish
	wg.Wait()
}
