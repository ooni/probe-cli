package measurexlite

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDependencies(t *testing.T) {
	t.Run("NewDialerWithoutResolver", func(t *testing.T) {
		t.Run("with nonnil dependences and hijacking", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			d := &dependencies{
				newDialerWithoutResolver: func(dl model.DebugLogger) model.Dialer {
					return &mocks.Dialer{
						MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
							return nil, mockedErr
						},
					}
				},
			}
			dialer := d.NewDialerWithoutResolver(model.DiscardLogger)
			ctx := context.Background()
			conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
			if !errors.Is(err, mockedErr) {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("with nonnil dependences and without hijacking", func(t *testing.T) {
			d := &dependencies{
				newDialerWithoutResolver: nil,
			}
			dialer := d.NewDialerWithoutResolver(model.DiscardLogger)
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // fail immediately
			conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
			if err == nil || err.Error() != netxlite.FailureInterrupted {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})

		t.Run("with nil dependences", func(t *testing.T) {
			var d *dependencies
			dialer := d.NewDialerWithoutResolver(model.DiscardLogger)
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // fail immediately
			conn, err := dialer.DialContext(ctx, "tcp", "1.1.1.1:443")
			if err == nil || err.Error() != netxlite.FailureInterrupted {
				t.Fatal("unexpected err", err)
			}
			if conn != nil {
				t.Fatal("expected nil conn")
			}
		})
	})
}
