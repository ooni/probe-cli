package netxlite

import (
	"crypto/tls"
	"errors"
	"net/http"
	"testing"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	nlmocks "github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestHTTP3Transport(t *testing.T) {
	t.Run("CloseIdleConnections", func(t *testing.T) {
		var (
			calledHTTP3  bool
			calledDialer bool
		)
		txp := &http3Transport{
			child: &nlmocks.HTTP3RoundTripper{
				MockClose: func() error {
					calledHTTP3 = true
					return nil
				},
			},
			dialer: &mocks.QUICDialer{
				MockCloseIdleConnections: func() {
					calledDialer = true
				},
			},
		}
		txp.CloseIdleConnections()
		if !calledHTTP3 || !calledDialer {
			t.Fatal("not called")
		}
	})

	t.Run("Network", func(t *testing.T) {
		txp := &http3Transport{}
		if txp.Network() != "udp" {
			t.Fatal("unexpected .Network return value")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		expected := errors.New("mocked error")
		txp := &http3Transport{
			child: &nlmocks.HTTP3RoundTripper{
				MockRoundTrip: func(req *http.Request) (*http.Response, error) {
					return nil, expected
				},
			},
		}
		resp, err := txp.RoundTrip(&http.Request{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("unexpected resp")
		}
	})
}

// verifyTypeChainForHTTP3 helps to verify type chains for HTTP3.
//
// Arguments:
//
// - t is the MANDATORY testing ref;
//
// - txp is the MANDATORY HTTP transport to verify;
//
// - underlyingLogger is the MANDATORY logger we expect to find;
//
// - qd is the OPTIONAL QUIC dialer: if not nil, we expect to
// see this value as the QUIC dialer, otherwise we will check the
// type chain of the real dialer;
//
// - config is the MANDATORY TLS config: we'll always check
// whether the TLSClientConfig is equal to this value: passing
// nil here means we expect to see nil in the object;
//
// - reso is the OPTIONAL resolver: if present and the qd is
// nil, we'll unwrap the QUIC dialer and check whether we have
// this resolver as the underlying resolver.
func verifyTypeChainForHTTP3(t *testing.T, txp model.HTTPTransport,
	underlyingLogger model.DebugLogger, qd model.QUICDialer,
	config *tls.Config, reso model.Resolver) {
	logger := txp.(*httpTransportLogger)
	if logger.Logger != underlyingLogger {
		t.Fatal("invalid logger")
	}
	ew := logger.HTTPTransport.(*httpTransportErrWrapper)
	h3txp := ew.HTTPTransport.(*http3Transport)
	if qd != nil && h3txp.dialer != qd {
		t.Fatal("invalid dialer")
	}
	if qd == nil {
		qdlog := h3txp.dialer.(*quicDialerLogger)
		qdr := qdlog.Dialer.(*quicDialerResolver)
		if reso != nil && qdr.Resolver != reso {
			t.Fatal("invalid resolver")
		}
	}
	h3 := h3txp.child.(*http3.RoundTripper)
	if h3.Dial == nil {
		t.Fatal("invalid Dial")
	}
	if !h3.DisableCompression {
		t.Fatal("invalid DisableCompression")
	}
	if h3.TLSClientConfig != config {
		t.Fatal("invalid TLSClientConfig")
	}
}

func TestNewHTTP3Transport(t *testing.T) {
	t.Run("creates the correct type chain", func(t *testing.T) {
		qd := &mocks.QUICDialer{}
		config := &tls.Config{}
		txp := NewHTTP3Transport(model.DiscardLogger, qd, config)
		verifyTypeChainForHTTP3(t, txp, model.DiscardLogger, qd, config, nil)
	})
}

func TestNewHTTP3TransportStdlib(t *testing.T) {
	t.Run("creates the correct type chain", func(t *testing.T) {
		txp := NewHTTP3TransportStdlib(model.DiscardLogger)
		verifyTypeChainForHTTP3(t, txp, model.DiscardLogger, nil, nil, nil)
	})
}

func TestNewHTTP3TransportWithResolver(t *testing.T) {
	t.Run("creates the correct type chain", func(t *testing.T) {
		reso := &mocks.Resolver{}
		txp := NewHTTP3TransportWithResolver(model.DiscardLogger, reso)
		verifyTypeChainForHTTP3(t, txp, model.DiscardLogger, nil, nil, reso)
	})
}

func TestNewHTTP3ClientWithResolver(t *testing.T) {
	reso := &mocks.Resolver{}
	clnt := NewHTTP3ClientWithResolver(model.DiscardLogger, reso)
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
	txpCc := txpEwrap.HTTPTransport.(*http3Transport)
	dialerLogger := txpCc.dialer.(*quicDialerLogger)
	dialerReso := dialerLogger.Dialer.(*quicDialerResolver)
	dialerLoggerInner := dialerReso.Dialer.(*quicDialerLogger)
	dialerWrapper := dialerLoggerInner.Dialer.(*quicDialerErrWrapper)
	dialerCompleter := dialerWrapper.QUICDialer.(*quicDialerHandshakeCompleter)
	dialerQUICGo := dialerCompleter.Dialer.(*quicDialerQUICGo)

	if dialerReso.Resolver != reso {
		t.Fatal("invalid resolver")
	}
	if dialerQUICGo.QUICListener == nil {
		t.Fatal("QUICListener should not be nil")
	}
}
