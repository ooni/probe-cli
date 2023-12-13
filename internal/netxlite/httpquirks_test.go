package netxlite

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewHTTPTransportWithResolver(t *testing.T) {
	expected := errors.New("mocked error")
	reso := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, expected
		},
	}
	txp := NewHTTPTransportWithResolver(model.DiscardLogger, reso)
	req, err := http.NewRequest("GET", "http://x.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("unexpected err")
	}
	if resp != nil {
		t.Fatal("expected nil resp")
	}
}

func TestNewHTTPTransport(t *testing.T) {
	t.Run("works as intended with failing dialer", func(t *testing.T) {
		called := &atomic.Int64{}
		expected := errors.New("mocked error")
		d := &dialerResolverWithTracing{
			Dialer: &mocks.Dialer{
				MockDialContext: func(ctx context.Context,
					network, address string) (net.Conn, error) {
					return nil, expected
				},
				MockCloseIdleConnections: func() {
					called.Add(1)
				},
			},
			Resolver: NewStdlibResolver(log.Log),
		}
		td := NewTLSDialer(d, NewTLSHandshakerStdlib(log.Log))
		txp := NewHTTPTransport(log.Log, d, td)
		client := &http.Client{Transport: txp}
		resp, err := client.Get("https://8.8.4.4/robots.txt")
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected", err)
		}
		if resp != nil {
			t.Fatal("expected non-nil response here")
		}
		client.CloseIdleConnections()
		if called.Load() < 1 {
			t.Fatal("did not propagate CloseIdleConnections")
		}
	})

	t.Run("creates the correct type chain", func(t *testing.T) {
		d := &mocks.Dialer{}
		td := &mocks.TLSDialer{}
		txp := NewHTTPTransport(log.Log, d, td)
		logger := txp.(*httpTransportLogger)
		if logger.Logger != log.Log {
			t.Fatal("invalid logger")
		}
		errWrapper := logger.HTTPTransport.(*httpTransportErrWrapper)
		connectionsCloser := errWrapper.HTTPTransport.(*httpTransportConnectionsCloser)
		withReadTimeout := connectionsCloser.Dialer.(*httpDialerWithReadTimeout)
		if withReadTimeout.Dialer != d {
			t.Fatal("invalid dialer")
		}
		tlsWithReadTimeout := connectionsCloser.TLSDialer.(*httpTLSDialerWithReadTimeout)
		if tlsWithReadTimeout.TLSDialer != td {
			t.Fatal("invalid tls dialer")
		}
		stdlib := connectionsCloser.HTTPTransport.(*httpTransportStdlib)
		if !stdlib.StdlibTransport.ForceAttemptHTTP2 {
			t.Fatal("invalid ForceAttemptHTTP2")
		}
		if !stdlib.StdlibTransport.DisableCompression {
			t.Fatal("invalid DisableCompression")
		}
		if stdlib.StdlibTransport.MaxConnsPerHost != 1 {
			t.Fatal("invalid MaxConnPerHost")
		}
		if stdlib.StdlibTransport.DialTLSContext == nil {
			t.Fatal("invalid DialTLSContext")
		}
		if stdlib.StdlibTransport.DialContext == nil {
			t.Fatal("invalid DialContext")
		}
	})
}

func TestNewHTTPTransportStdlib(t *testing.T) {
	txp := NewHTTPTransportStdlib(log.Log)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately!
	req, err := http.NewRequestWithContext(ctx, "GET", "http://x.org", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
	if resp != nil {
		t.Fatal("unexpected resp")
	}
	if txp.Network() != "tcp" {
		t.Fatal("unexpected .Network return value")
	}
	txp.CloseIdleConnections()
}

func TestNewHTTPClientStdlib(t *testing.T) {
	clnt := NewHTTPClientStdlib(model.DiscardLogger)
	ewc, ok := clnt.(*httpClientErrWrapper)
	if !ok {
		t.Fatal("expected *httpClientErrWrapper")
	}
	_, ok = ewc.HTTPClient.(*http.Client)
	if !ok {
		t.Fatal("expected *http.Client")
	}
}

func TestNewHTTPClientWithResolver(t *testing.T) {
	reso := &mocks.Resolver{}
	clnt := NewHTTPClientWithResolver(model.DiscardLogger, reso)
	ewc, ok := clnt.(*httpClientErrWrapper)
	if !ok {
		t.Fatal("expected *httpClientErrWrapper")
	}
	httpClnt, ok := ewc.HTTPClient.(*http.Client)
	if !ok {
		t.Fatal("expected *http.Client")
	}
	txp := httpClnt.Transport.(*httpTransportLogger)
	txpEwrap := txp.HTTPTransport.(*httpTransportErrWrapper)
	txpCc := txpEwrap.HTTPTransport.(*httpTransportConnectionsCloser)
	dialer := txpCc.Dialer.(*httpDialerWithReadTimeout)
	dialerLogger := dialer.Dialer.(*dialerLogger)
	dialerReso := dialerLogger.Dialer.(*dialerResolverWithTracing)
	if dialerReso.Resolver != reso {
		t.Fatal("invalid resolver")
	}
}
