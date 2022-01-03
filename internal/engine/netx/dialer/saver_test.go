package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSaverDialerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := &trace.Saver{}
	dlr := &saverDialer{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, expected
			},
		},
		Saver: saver,
	}
	conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, expected) {
		t.Fatal("expected another error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	ev := saver.Read()
	if len(ev) != 1 {
		t.Fatal("expected a single event here")
	}
	if ev[0].Address != "www.google.com:443" {
		t.Fatal("unexpected Address")
	}
	if ev[0].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if !errors.Is(ev[0].Err, expected) {
		t.Fatal("unexpected Err")
	}
	if ev[0].Name != netxlite.ConnectOperation {
		t.Fatal("unexpected Name")
	}
	if ev[0].Proto != "tcp" {
		t.Fatal("unexpected Proto")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverConnDialerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := &trace.Saver{}
	dlr := &saverConnDialer{
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, expected
			},
		},
		Saver: saver,
	}
	conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSaverConnDialerSuccess(t *testing.T) {
	saver := &trace.Saver{}
	dlr := &saverConnDialer{
		Dialer: &saverDialer{
			Dialer: &mocks.Dialer{
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
			},
			Saver: saver,
		},
		Saver: saver,
	}
	conn, err := dlr.DialContext(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal("not the error we expected", err)
	}
	conn.Read(nil)
	conn.Write(nil)
	conn.Close()
	events := saver.Read()
	if len(events) != 3 {
		t.Fatal("unexpected number of events saved", len(events))
	}
	if events[0].Name != "connect" {
		t.Fatal("expected a connect event")
	}
	saverCheckConnectEvent(t, &events[0])
	if events[1].Name != "read" {
		t.Fatal("expected a read event")
	}
	saverCheckReadEvent(t, &events[1])
	if events[2].Name != "write" {
		t.Fatal("expected a write event")
	}
	saverCheckWriteEvent(t, &events[2])
}

func saverCheckConnectEvent(t *testing.T, ev *trace.Event) {
	// TODO(bassosimone): implement
}

func saverCheckReadEvent(t *testing.T, ev *trace.Event) {
	// TODO(bassosimone): implement
}

func saverCheckWriteEvent(t *testing.T, ev *trace.Event) {
	// TODO(bassosimone): implement
}
