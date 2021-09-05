package iox

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
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
	r := &mocks.Reader{
		MockRead: func(b []byte) (int, error) {
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
	r := &mocks.Reader{
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

func TestCopyContextCommonCase(t *testing.T) {
	r := strings.NewReader("deadbeef")
	ctx := context.Background()
	out, err := CopyContext(ctx, io.Discard, r)
	if err != nil {
		t.Fatal(err)
	}
	if out != 8 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestCopyContextWithError(t *testing.T) {
	expected := errors.New("mocked error")
	r := &mocks.Reader{
		MockRead: func(b []byte) (int, error) {
			return 0, expected
		},
	}
	ctx := context.Background()
	out, err := CopyContext(ctx, io.Discard, r)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if out != 0 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestCopyContextWithCancelledContext(t *testing.T) {
	r := strings.NewReader("deadbeef")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := CopyContext(ctx, io.Discard, r)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if out != 0 {
		t.Fatal("not the expected number of bytes")
	}
}

func TestCopyContextWithErrorAndCancelledContext(t *testing.T) {
	expected := errors.New("mocked error")
	r := &mocks.Reader{
		MockRead: func(b []byte) (int, error) {
			time.Sleep(time.Millisecond)
			return 0, expected
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := CopyContext(ctx, io.Discard, r)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if out != 0 {
		t.Fatal("not the expected number of bytes")
	}
}
