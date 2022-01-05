package runtimex_test

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestPanicOnError(t *testing.T) {
	badfunc := func(in error) (out error) {
		defer func() {
			out = recover().(error)
		}()
		runtimex.PanicOnError(in, "antani failed")
		return
	}

	t.Run("error is nil", func(t *testing.T) {
		runtimex.PanicOnError(nil, "antani failed")
	})

	t.Run("error is not nil", func(t *testing.T) {
		expected := errors.New("mocked error")
		if !errors.Is(badfunc(expected), expected) {
			t.Fatal("not the error we expected")
		}
	})
}

func TestPanicIfFalse(t *testing.T) {
	badfunc := func(in bool, message string) (out error) {
		defer func() {
			out = errors.New(recover().(string))
		}()
		runtimex.PanicIfFalse(in, message)
		return
	}

	t.Run("assertion is true", func(t *testing.T) {
		runtimex.PanicIfFalse(true, "antani failed")
	})

	t.Run("assertion is false", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(false, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}

func TestPanicIfTrue(t *testing.T) {
	badfunc := func(in bool, message string) (out error) {
		defer func() {
			out = errors.New(recover().(string))
		}()
		runtimex.PanicIfTrue(in, message)
		return
	}

	t.Run("assertion is false", func(t *testing.T) {
		runtimex.PanicIfTrue(false, "antani failed")
	})

	t.Run("assertion is true", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(true, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}

func TestPanicIfNil(t *testing.T) {
	badfunc := func(in interface{}, message string) (out error) {
		defer func() {
			out = errors.New(recover().(string))
		}()
		runtimex.PanicIfNil(in, message)
		return
	}

	t.Run("value is not nil", func(t *testing.T) {
		runtimex.PanicIfNil(false, "antani failed")
	})

	t.Run("value is nil", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(nil, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}
