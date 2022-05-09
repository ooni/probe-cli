package httptransport_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewHTTP3Transport(t *testing.T) {
	// make sure we can create a working transport using this factory.
	expected := errors.New("mocked error")
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		QUICDialer: &mocks.QUICDialer{
			MockDialContext: func(ctx context.Context, network, address string,
				tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return nil, expected
			},
			MockCloseIdleConnections: func() {
				// nothing
			},
		},
	})
	req, err := http.NewRequest("GET", "https://google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("unexpected err", err)
	}
	if resp != nil {
		t.Fatal("expected nil resp")
	}
}
