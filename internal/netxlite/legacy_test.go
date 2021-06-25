package netxlite

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

func TestReduceErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := reduceErrors(nil)
		if result != nil {
			t.Fatal("wrong result")
		}
	})
	t.Run("single error", func(t *testing.T) {
		err := errors.New("mocked error")
		result := reduceErrors([]error{err})
		if result != err {
			t.Fatal("wrong result")
		}
	})
	t.Run("multiple errors", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := errors.New("mocked error #2")
		result := reduceErrors([]error{err1, err2})
		if result.Error() != "mocked error #1" {
			t.Fatal("wrong result")
		}
	})
	t.Run("multiple errors with meaningful ones", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := &errorx.ErrWrapper{
			Failure: "unknown_failure: antani",
		}
		err3 := &errorx.ErrWrapper{
			Failure: errorx.FailureConnectionRefused,
		}
		err4 := errors.New("mocked error #3")
		result := reduceErrors([]error{err1, err2, err3, err4})
		if result.Error() != errorx.FailureConnectionRefused {
			t.Fatal("wrong result")
		}
	})
}
