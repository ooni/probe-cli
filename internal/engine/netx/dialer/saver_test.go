package dialer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

func TestSaverDialerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := &trace.Saver{}
	dlr := dialer.SaverDialer{
		Dialer: dialer.FakeDialer{
			Err: expected,
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
	if ev[0].Name != errorx.ConnectOperation {
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
	dlr := dialer.SaverConnDialer{
		Dialer: dialer.FakeDialer{
			Err: expected,
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
