package tlsdialer_test

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
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

func TestTLSDialerFailureSplitHostPort(t *testing.T) {
	dialer := tlsdialer.TLSDialer{}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com") // missing port
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureDialing(t *testing.T) {
	dialer := tlsdialer.TLSDialer{Dialer: tlsdialer.EOFDialer{}}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}

func TestTLSDialerFailureHandshaking(t *testing.T) {
	rec := &RecorderTLSHandshaker{TLSHandshaker: &netxlite.TLSHandshakerConfigurable{}}
	dialer := tlsdialer.TLSDialer{
		Dialer:        tlsdialer.EOFConnDialer{},
		TLSHandshaker: rec,
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
	if rec.SNI != "www.google.com" {
		t.Fatal("unexpected SNI value")
	}
}

func TestTLSDialerFailureHandshakingOverrideSNI(t *testing.T) {
	rec := &RecorderTLSHandshaker{TLSHandshaker: &netxlite.TLSHandshakerConfigurable{}}
	dialer := tlsdialer.TLSDialer{
		Config: &tls.Config{
			ServerName: "x.org",
		},
		Dialer:        tlsdialer.EOFConnDialer{},
		TLSHandshaker: rec,
	}
	conn, err := dialer.DialTLSContext(
		context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
	if rec.SNI != "x.org" {
		t.Fatal("unexpected SNI value")
	}
}

type RecorderTLSHandshaker struct {
	tlsdialer.TLSHandshaker
	SNI string
}

func (h *RecorderTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	h.SNI = config.ServerName
	return h.TLSHandshaker.Handshake(ctx, conn, config)
}

func TestDialTLSContextGood(t *testing.T) {
	dialer := tlsdialer.TLSDialer{
		Config:        &tls.Config{ServerName: "google.com"},
		Dialer:        new(net.Dialer),
		TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
	}
	conn, err := dialer.DialTLSContext(context.Background(), "tcp", "google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("connection is nil")
	}
	conn.Close()
}

func TestUTLSHandshakerChrome(t *testing.T) {
	dialer := tlsdialer.TLSDialer{
		Config: &tls.Config{ServerName: "google.com"},
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.UTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
			ClientHelloID: &utls.HelloChrome_Auto,
		},
	}
	conn, err := dialer.DialTLSContext(context.Background(), "tcp", "google.com:443")
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if conn == nil {
		t.Fatal("nil connection")
	}
}
