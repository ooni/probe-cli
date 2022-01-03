package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	nlmocks "github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestHTTP3Dialer(t *testing.T) {
	t.Run("Dial", func(t *testing.T) {
		expected := errors.New("mocked error")
		d := &http3Dialer{
			QUICDialer: &mocks.QUICDialer{
				MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
					return nil, expected
				},
			},
		}
		sess, err := d.dial("", "", &tls.Config{}, &quic.Config{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if sess != nil {
			t.Fatal("unexpected resp")
		}
	})
}

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

func TestNewHTTP3Transport(t *testing.T) {
	t.Run("creates the correct type chain", func(t *testing.T) {
		qd := &mocks.QUICDialer{}
		config := &tls.Config{}
		txp := NewHTTP3Transport(log.Log, qd, config)
		logger := txp.(*httpTransportLogger)
		if logger.Logger != log.Log {
			t.Fatal("invalid logger")
		}
		h3txp := logger.HTTPTransport.(*http3Transport)
		if h3txp.dialer != qd {
			t.Fatal("invalid dialer")
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
	})
}
