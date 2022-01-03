package netx

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestEmitterFailure(t *testing.T) {
	ctx := context.Background()
	saver := &handlers.SavingHandler{}
	ctx = modelx.WithMeasurementRoot(ctx, &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   saver,
	})
	d := EmitterDialer{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected a nil conn here")
	}
	events := saver.Read()
	if len(events) != 1 {
		t.Fatal("unexpected number of events saved")
	}
	if events[0].Connect == nil {
		t.Fatal("expected non nil Connect")
	}
	conninfo := events[0].Connect
	emitterCheckConnectEventCommon(t, conninfo, io.EOF)
}

func emitterCheckConnectEventCommon(
	t *testing.T, conninfo *modelx.ConnectEvent, err error) {
	if conninfo.DurationSinceBeginning == 0 {
		t.Fatal("unexpected DurationSinceBeginning value")
	}
	if !errors.Is(conninfo.Error, err) {
		t.Fatal("unexpected Error value")
	}
	if conninfo.Network != "tcp" {
		t.Fatal("unexpected Network value")
	}
	if conninfo.RemoteAddress != "www.google.com:443" {
		t.Fatal("unexpected Network value")
	}
	if conninfo.SyscallDuration == 0 {
		t.Fatal("unexpected SyscallDuration value")
	}
}

func TestEmitterSuccess(t *testing.T) {
	ctx := context.Background()
	saver := &handlers.SavingHandler{}
	ctx = modelx.WithMeasurementRoot(ctx, &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   saver,
	})
	d := EmitterDialer{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return &mocks.Conn{
				MockRead: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockWrite: func(b []byte) (int, error) {
					return 0, io.EOF
				},
				MockClose: func() error {
					return io.EOF
				},
				MockLocalAddr: func() net.Addr {
					return &net.TCPAddr{Port: 12345}
				},
			}, nil
		},
	}}
	conn, err := d.DialContext(ctx, "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal("we expected no error")
	}
	if conn == nil {
		t.Fatal("expected a non-nil conn here")
	}
	conn.Read(nil)
	conn.Write(nil)
	conn.Close()
	events := saver.Read()
	if len(events) != 4 {
		t.Fatal("unexpected number of events saved")
	}
	if events[0].Connect == nil {
		t.Fatal("expected non nil Connect")
	}
	conninfo := events[0].Connect
	emitterCheckConnectEventCommon(t, conninfo, nil)
	if events[1].Read == nil {
		t.Fatal("expected non nil Read")
	}
	emitterCheckReadEvent(t, events[1].Read)
	if events[2].Write == nil {
		t.Fatal("expected non nil Write")
	}
	emitterCheckWriteEvent(t, events[2].Write)
	if events[3].Close == nil {
		t.Fatal("expected non nil Close")
	}
	emitterCheckCloseEvent(t, events[3].Close)
}

func emitterCheckReadEvent(t *testing.T, ev *modelx.ReadEvent) {
	if ev.DurationSinceBeginning == 0 {
		t.Fatal("unexpected DurationSinceBeginning")
	}
	if !errors.Is(ev.Error, io.EOF) {
		t.Fatal("unexpected Error")
	}
	if ev.NumBytes != 0 {
		t.Fatal("unexpected NumBytes")
	}
	if ev.SyscallDuration == 0 {
		t.Fatal("unexpected SyscallDuration")
	}
}

func emitterCheckWriteEvent(t *testing.T, ev *modelx.WriteEvent) {
	if ev.DurationSinceBeginning == 0 {
		t.Fatal("unexpected DurationSinceBeginning")
	}
	if !errors.Is(ev.Error, io.EOF) {
		t.Fatal("unexpected Error")
	}
	if ev.NumBytes != 0 {
		t.Fatal("unexpected NumBytes")
	}
	if ev.SyscallDuration == 0 {
		t.Fatal("unexpected SyscallDuration")
	}
}

func emitterCheckCloseEvent(t *testing.T, ev *modelx.CloseEvent) {
	if ev.DurationSinceBeginning == 0 {
		t.Fatal("unexpected DurationSinceBeginning")
	}
	if !errors.Is(ev.Error, io.EOF) {
		t.Fatal("unexpected Error")
	}
	if ev.SyscallDuration == 0 {
		t.Fatal("unexpected SyscallDuration")
	}
}
