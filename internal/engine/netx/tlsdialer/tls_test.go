package tlsdialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
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

func TestErrorWrapperTLSHandshakerFailure(t *testing.T) {
	h := tlsdialer.ErrorWrapperTLSHandshaker{TLSHandshaker: tlsdialer.EOFTLSHandshaker{}}
	conn, _, err := h.Handshake(
		context.Background(), tlsdialer.EOFConn{}, new(tls.Config))
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	var errWrapper *errorx.ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot cast to ErrWrapper")
	}
	if errWrapper.Failure != errorx.FailureEOFError {
		t.Fatal("unexpected Failure")
	}
	if errWrapper.Operation != errorx.TLSHandshakeOperation {
		t.Fatal("unexpected Operation")
	}
}

func TestEmitterTLSHandshakerFailure(t *testing.T) {
	saver := &handlers.SavingHandler{}
	ctx := modelx.WithMeasurementRoot(context.Background(), &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   saver,
	})
	h := tlsdialer.EmitterTLSHandshaker{TLSHandshaker: tlsdialer.EOFTLSHandshaker{}}
	conn, _, err := h.Handshake(ctx, tlsdialer.EOFConn{}, &tls.Config{
		ServerName: "www.kernel.org",
	})
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	events := saver.Read()
	if len(events) != 2 {
		t.Fatal("Wrong number of events")
	}
	if events[0].TLSHandshakeStart == nil {
		t.Fatal("missing TLSHandshakeStart event")
	}
	if events[0].TLSHandshakeStart.DurationSinceBeginning == 0 {
		t.Fatal("expected nonzero DurationSinceBeginning")
	}
	if events[0].TLSHandshakeStart.SNI != "www.kernel.org" {
		t.Fatal("expected nonzero SNI")
	}
	if events[1].TLSHandshakeDone == nil {
		t.Fatal("missing TLSHandshakeDone event")
	}
	if events[1].TLSHandshakeDone.DurationSinceBeginning == 0 {
		t.Fatal("expected nonzero DurationSinceBeginning")
	}
}
