package netxlite

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestReadAllContext(t *testing.T) {
	t.Run("with success and background context", func(t *testing.T) {
		r := strings.NewReader("deadbeef")
		ctx := context.Background()
		out, err := ReadAllContext(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 8 {
			t.Fatal("not the expected number of bytes")
		}
	})

	t.Run("with success and wrapped io.EOF", func(t *testing.T) {
		// See https://github.com/ooni/probe/issues/1965
		wg := &sync.WaitGroup{}
		wg.Add(1)
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				// "When Read encounters an error or end-of-file condition
				// after successfully reading n > 0 bytes, it returns
				// the number of bytes read. It may return the (non-nil)
				// error from the same call or return the error (and n == 0)
				// from a subsequent call.""
				//
				// See https://pkg.go.dev/io#Reader
				//
				// Note: Returning a wrapped error to ensure we address
				// https://github.com/ooni/probe/issues/1965
				return len(b), NewErrWrapper(ClassifyGenericError,
					ReadOperation, io.EOF)
			},
		}
		out, err := ReadAllContext(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) <= 0 {
			t.Fatal("we expected to see a positive number of bytes here")
		}
		wg.Wait()
	})

	t.Run("with failure and background context", func(t *testing.T) {
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
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if len(out) != 0 {
			t.Fatal("not the expected number of bytes")
		}
	})

	t.Run("with success and cancelled context", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		sigch := make(chan interface{})
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				<-sigch
				// "When Read encounters an error or end-of-file condition
				// after successfully reading n > 0 bytes, it returns
				// the number of bytes read. It may return the (non-nil)
				// error from the same call or return the error (and n == 0)
				// from a subsequent call.""
				//
				// See https://pkg.go.dev/io#Reader
				return len(b), io.EOF
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		out, err := ReadAllContext(ctx, r)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("not the error we expected", err)
		}
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if len(out) != 0 {
			t.Fatal("not the expected number of bytes")
		}
		close(sigch)
		wg.Wait()
	})

	t.Run("with failure and cancelled context", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		sigch := make(chan interface{})
		expected := errors.New("mocked error")
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				<-sigch
				return 0, expected
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		out, err := ReadAllContext(ctx, r)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("not the error we expected", err)
		}
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if len(out) != 0 {
			t.Fatal("not the expected number of bytes")
		}
		close(sigch)
		wg.Wait()
	})
}

func TestCopyContext(t *testing.T) {
	t.Run("with success and background context", func(t *testing.T) {
		r := strings.NewReader("deadbeef")
		ctx := context.Background()
		out, err := CopyContext(ctx, io.Discard, r)
		if err != nil {
			t.Fatal(err)
		}
		if out != 8 {
			t.Fatal("not the expected number of bytes")
		}
	})

	t.Run("with success and wrapped io.EOF", func(t *testing.T) {
		// See https://github.com/ooni/probe/issues/1965
		wg := &sync.WaitGroup{}
		wg.Add(1)
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				// "When Read encounters an error or end-of-file condition
				// after successfully reading n > 0 bytes, it returns
				// the number of bytes read. It may return the (non-nil)
				// error from the same call or return the error (and n == 0)
				// from a subsequent call.""
				//
				// See https://pkg.go.dev/io#Reader
				//
				// Note: Returning a wrapped error to ensure we address
				// https://github.com/ooni/probe/issues/1965
				return len(b), NewErrWrapper(ClassifyGenericError,
					ReadOperation, io.EOF)
			},
		}
		out, err := CopyContext(context.Background(), io.Discard, r)
		if err != nil {
			t.Fatal(err)
		}
		if out <= 0 {
			t.Fatal("we expected to see a positive number of bytes here")
		}
		wg.Wait()
	})

	t.Run("with failure and background context", func(t *testing.T) {
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
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if out != 0 {
			t.Fatal("not the expected number of bytes")
		}
	})

	t.Run("with success and cancelled context", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		sigch := make(chan interface{})
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				<-sigch
				// "When Read encounters an error or end-of-file condition
				// after successfully reading n > 0 bytes, it returns
				// the number of bytes read. It may return the (non-nil)
				// error from the same call or return the error (and n == 0)
				// from a subsequent call.""
				//
				// See https://pkg.go.dev/io#Reader
				return len(b), io.EOF
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		out, err := CopyContext(ctx, io.Discard, r)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("not the error we expected", err)
		}
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if out != 0 {
			t.Fatal("not the expected number of bytes")
		}
		close(sigch)
		wg.Wait()
	})

	t.Run("with failure and cancelled context", func(t *testing.T) {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		sigch := make(chan interface{})
		expected := errors.New("mocked error")
		r := &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				defer wg.Done()
				<-sigch
				return 0, expected
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		out, err := CopyContext(ctx, io.Discard, r)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("not the error we expected", err)
		}
		var errWrapper *ErrWrapper
		if !errors.As(err, &errWrapper) {
			t.Fatal("the returned error is not wrapped")
		}
		if out != 0 {
			t.Fatal("not the expected number of bytes")
		}
		close(sigch)
		wg.Wait()
	})
}
