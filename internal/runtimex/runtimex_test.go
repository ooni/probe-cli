package runtimex

import (
	"errors"
	"testing"
)

func TestPanicOnError(t *testing.T) {
	badfunc := func(in error) (out error) {
		defer func() {
			out = recover().(error)
		}()
		PanicOnError(in, "we expect this assertion to fail")
		return
	}

	t.Run("error is nil", func(t *testing.T) {
		PanicOnError(nil, "this assertion should not fail")
	})

	t.Run("error is not nil", func(t *testing.T) {
		expected := errors.New("mocked error")
		if !errors.Is(badfunc(expected), expected) {
			t.Fatal("not the error we expected")
		}
	})
}

func TestAssert(t *testing.T) {
	badfunc := func(in bool, message string) (out error) {
		defer func() {
			out = recover().(error)
		}()
		Assert(in, message)
		return
	}

	t.Run("assertion is true", func(t *testing.T) {
		Assert(true, "this assertion should not fail")
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
			out = recover().(error)
		}()
		PanicIfTrue(in, message)
		return
	}

	t.Run("assertion is false", func(t *testing.T) {
		PanicIfTrue(false, "this assertion should not fail")
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
			out = recover().(error)
		}()
		PanicIfNil(in, message)
		return
	}

	t.Run("value is not nil", func(t *testing.T) {
		PanicIfNil(false, "this assertion should not fail")
	})

	t.Run("value is nil", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(nil, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}
