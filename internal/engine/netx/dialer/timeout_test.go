package dialer_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
)

type SlowDialer struct{}

func (SlowDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(30 * time.Second):
		return nil, io.EOF
	}
}

func TestTimeoutDialer(t *testing.T) {
	d := dialer.TimeoutDialer{Dialer: SlowDialer{}, ConnectTimeout: time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
