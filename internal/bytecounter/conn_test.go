package bytecounter

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
)

func TestConnWorksOnSuccess(t *testing.T) {
	counter := New()
	underlying := &mockablex.Conn{
		MockRead: func(b []byte) (int, error) {
			return 10, nil
		},
		MockWrite: func(b []byte) (int, error) {
			return 4, nil
		},
	}
	conn := &Conn{
		Conn:    underlying,
		Counter: counter,
	}
	if _, err := conn.Read(make([]byte, 128)); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if counter.BytesReceived() != 10 {
		t.Fatal("unexpected number of bytes received")
	}
	if counter.BytesSent() != 4 {
		t.Fatal("unexpected number of bytes sent")
	}
}

func TestConnWorksOnFailure(t *testing.T) {
	readError := errors.New("read error")
	writeError := errors.New("write error")
	counter := New()
	underlying := &mockablex.Conn{
		MockRead: func(b []byte) (int, error) {
			return 0, readError
		},
		MockWrite: func(b []byte) (int, error) {
			return 0, writeError
		},
	}
	conn := &Conn{
		Conn:    underlying,
		Counter: counter,
	}
	if _, err := conn.Read(make([]byte, 128)); !errors.Is(err, readError) {
		t.Fatal("not the error we expected", err)
	}
	if _, err := conn.Write(make([]byte, 1024)); !errors.Is(err, writeError) {
		t.Fatal("not the error we expected", err)
	}
	if counter.BytesReceived() != 0 {
		t.Fatal("unexpected number of bytes received")
	}
	if counter.BytesSent() != 0 {
		t.Fatal("unexpected number of bytes sent")
	}
}
