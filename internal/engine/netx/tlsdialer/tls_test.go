package tlsdialer_test

import (
	"context"
	"crypto/tls"
	"io"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSystemTLSHandshakerEOFError(t *testing.T) {
	h := &netxlite.TLSHandshakerConfigurable{}
	conn, _, err := h.Handshake(context.Background(), tlsdialer.EOFConn{}, &tls.Config{
		ServerName: "x.org",
	})
	if err != io.EOF {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
}

type SetDeadlineConn struct {
	tlsdialer.EOFConn
	deadlines []time.Time
}

func (c *SetDeadlineConn) SetDeadline(t time.Time) error {
	c.deadlines = append(c.deadlines, t)
	return nil
}
