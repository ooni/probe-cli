package mocks

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/lucas-clemente/quic-go"
)

func TestQUICContextDialer(t *testing.T) {
	t.Run("DialContext", func(t *testing.T) {
		expected := errors.New("mocked error")
		qcd := &QUICContextDialer{
			MockDialContext: func(ctx context.Context, network string, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
				return nil, expected
			},
		}
		ctx := context.Background()
		tlsConfig := &tls.Config{}
		quicConfig := &quic.Config{}
		sess, err := qcd.DialContext(ctx, "udp", "dns.google:443", tlsConfig, quicConfig)
		if !errors.Is(err, expected) {
			t.Fatal("not the error we expected")
		}
		if sess != nil {
			t.Fatal("expected nil session")
		}
	})
}
