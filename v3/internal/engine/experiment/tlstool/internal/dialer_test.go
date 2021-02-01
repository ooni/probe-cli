package internal_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlstool/internal"
)

func TestDialerFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dialer := internal.Dialer{Dialer: internal.FakeDialer{
		Err: expected,
	}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "8.8.8.8:853")
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestDialerSuccess(t *testing.T) {
	splitter := func([]byte) [][]byte {
		return nil // any value is fine we just a need a splitter != nil here
	}
	innerconn := &internal.FakeConn{}
	dialer := internal.Dialer{
		Delay:    12345,
		Dialer:   internal.FakeDialer{Conn: innerconn},
		Splitter: splitter,
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", "8.8.8.8:853")
	if err != nil {
		t.Fatal(err)
	}
	sconn, ok := conn.(internal.SplitterWriter)
	if !ok {
		t.Fatal("the outer connection is not a splitter")
	}
	if sconn.Splitter == nil {
		t.Fatal("not the splitter we expected")
	}
	dconn, ok := sconn.Conn.(internal.SleeperWriter)
	if !ok {
		t.Fatal("the inner connection is not a sleeper")
	}
	if dconn.Delay != 12345 {
		t.Fatal("invalid delay")
	}
	if dconn.Conn != innerconn {
		t.Fatal("invalid inner connection")
	}
}
