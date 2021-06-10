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
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
)

func TestSystemTLSHandshakerEOFError(t *testing.T) {
	h := tlsdialer.SystemTLSHandshaker{}
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

func TestTimeoutTLSHandshakerSetDeadlineError(t *testing.T) {
	h := tlsdialer.TimeoutTLSHandshaker{
		TLSHandshaker:    tlsdialer.SystemTLSHandshaker{},
		HandshakeTimeout: 200 * time.Millisecond,
	}
	expected := errors.New("mocked error")
	conn, _, err := h.Handshake(
		context.Background(), &tlsdialer.FakeConn{SetDeadlineError: expected},
		new(tls.Config))
	if !errors.Is(err, expected) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
}

func TestTimeoutTLSHandshakerEOFError(t *testing.T) {
	h := tlsdialer.TimeoutTLSHandshaker{
		TLSHandshaker:    tlsdialer.SystemTLSHandshaker{},
		HandshakeTimeout: 200 * time.Millisecond,
	}
	conn, _, err := h.Handshake(
		context.Background(), tlsdialer.EOFConn{}, &tls.Config{ServerName: "x.org"})
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
}

func TestTimeoutTLSHandshakerCallsSetDeadline(t *testing.T) {
	h := tlsdialer.TimeoutTLSHandshaker{
		TLSHandshaker:    tlsdialer.SystemTLSHandshaker{},
		HandshakeTimeout: 200 * time.Millisecond,
	}
	underlying := &SetDeadlineConn{}
	conn, _, err := h.Handshake(
		context.Background(), underlying, &tls.Config{ServerName: "x.org"})
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	if len(underlying.deadlines) != 2 {
		t.Fatal("SetDeadline not called twice")
	}
	if underlying.deadlines[0].Before(time.Now()) {
		t.Fatal("the first SetDeadline call was incorrect")
	}
	if !underlying.deadlines[1].IsZero() {
		t.Fatal("the second SetDeadline call was incorrect")
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
	if !errors.As(err, &io.EOF) {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	var handshakeErr tlsdialer.ErrTLSHandshake
	if !errors.As(err, &handshakeErr) {
		t.Fatal("cannot cast to tlsdialer.ErrTLSHandshake")
	}
	if *archival.NewFailure(err) != errorx.FailureEOFError {
		t.Fatal("unexpected Failure")
	}
	if *archival.NewFailedOperation(err) != errorx.TLSHandshakeOperation {
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
	if events[0].TLSHandshakeStart.ConnID == 0 {
		t.Fatal("expected nonzero ConnID")
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
	if events[1].TLSHandshakeDone.ConnID == 0 {
		t.Fatal("expected nonzero ConnID")
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
	rec := &RecorderTLSHandshaker{TLSHandshaker: tlsdialer.SystemTLSHandshaker{}}
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
	rec := &RecorderTLSHandshaker{TLSHandshaker: tlsdialer.SystemTLSHandshaker{}}
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
		TLSHandshaker: tlsdialer.SystemTLSHandshaker{},
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

func TestDialTLSContextTimeout(t *testing.T) {
	dialer := tlsdialer.TLSDialer{
		Config: &tls.Config{ServerName: "google.com"},
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.ErrorWrapperTLSHandshaker{
			TLSHandshaker: tlsdialer.TimeoutTLSHandshaker{
				TLSHandshaker:    tlsdialer.SystemTLSHandshaker{},
				HandshakeTimeout: 10 * time.Microsecond,
			},
		},
	}
	conn, err := dialer.DialTLSContext(context.Background(), "tcp", "google.com:443")
	if *archival.NewFailure(err) != errorx.FailureGenericTimeoutError {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("connection is not nil")
	}
}
