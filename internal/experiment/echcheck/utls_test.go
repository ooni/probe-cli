package echcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	utls "gitlab.com/yawning/utls.git"
)

func TestTLSHandshakerWithExtension(t *testing.T) {
	t.Run("when the TLS handshake fails", func(t *testing.T) {
		thx := &tlsHandshakerWithExtensions{
			extensions: []utls.TLSExtension{},
			dl:         model.DiscardLogger,
			id:         &utls.HelloChrome_70,
		}

		expected := errors.New("mocked error")
		tcpConn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, expected
			},
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}

		tlsConn, err := thx.Handshake(context.Background(), tcpConn, tlsConfig)
		if !errors.Is(err, expected) {
			t.Fatal(err)
		}
		if tlsConn != nil {
			t.Fatal("expected nil tls conn")
		}
	})
}
