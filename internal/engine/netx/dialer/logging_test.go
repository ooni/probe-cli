package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
)

func TestLoggingDialerFailure(t *testing.T) {
	d := &loggingDialer{
		Dialer: mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, io.EOF
			},
		},
		Logger: log.Log,
	}
	conn, err := d.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
