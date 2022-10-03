package tlsmiddlebox

import (
	"context"
	"io"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDialerWithTTL(t *testing.T) {
	t.Run("DialContext on success", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedConn := &mocks.Conn{}
			d := &dialerTTLWrapper{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return expectedConn, nil
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "", "")
			if err != nil {
				t.Fatal(err)
			}
			errWrapperConn := conn.(*dialerTTLWrapperConn)
			if errWrapperConn.Conn != expectedConn {
				t.Fatal("unexpected conn")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			d := &dialerTTLWrapper{
				Dialer: &mocks.Dialer{
					MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
						return nil, expectedErr
					},
				},
			}
			ctx := context.Background()
			conn, err := d.DialContext(ctx, "", "")
			if err == nil || err.Error() != netxlite.FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})
}
