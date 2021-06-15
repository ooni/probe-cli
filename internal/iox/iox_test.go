package iox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReadAllContextCommonCase(t *testing.T) {
	r := strings.NewReader("deadbeef")
	ctx := context.Background()
	out, err := ReadAllContext(ctx, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 8 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestReadAllContextWithError(t *testing.T) {
	expected := errors.New("mocked error")
	r := &MockableReader{
		MockRead: func(b []byte) (int, error) {
			time.Sleep(time.Millisecond)
			return 0, expected
		},
	}
	ctx := context.Background()
	out, err := ReadAllContext(ctx, r)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if len(out) != 0 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestReadAllContextWithCancelledContext(t *testing.T) {
	r := strings.NewReader("deadbeef")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := ReadAllContext(ctx, r)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if len(out) != 0 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestReadAllContextWithErrorAndCancelledContext(t *testing.T) {
	expected := errors.New("mocked error")
	r := &MockableReader{
		MockRead: func(b []byte) (int, error) {
			time.Sleep(time.Millisecond)
			return 0, expected
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := ReadAllContext(ctx, r)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if len(out) != 0 {
		t.Fatal("not the expected number of bytes")
	}
}
