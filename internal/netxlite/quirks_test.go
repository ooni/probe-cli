package netxlite

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

func TestQuirkReduceErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := quirkReduceErrors(nil)
		if result != nil {
			t.Fatal("wrong result")
		}
	})

	t.Run("single error", func(t *testing.T) {
		err := errors.New("mocked error")
		result := quirkReduceErrors([]error{err})
		if result != err {
			t.Fatal("wrong result")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := errors.New("mocked error #2")
		result := quirkReduceErrors([]error{err1, err2})
		if result.Error() != "mocked error #1" {
			t.Fatal("wrong result")
		}
	})

	t.Run("multiple errors with meaningful ones", func(t *testing.T) {
		err1 := errors.New("mocked error #1")
		err2 := errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError,
			errorsx.CloseOperation,
			errors.New("antani"),
		)
		err3 := errorsx.NewErrWrapper(
			errorsx.ClassifyGenericError,
			errorsx.CloseOperation,
			errorsx.ECONNREFUSED,
		)
		err4 := errors.New("mocked error #3")
		result := quirkReduceErrors([]error{err1, err2, err3, err4})
		if result.Error() != errorsx.FailureConnectionRefused {
			t.Fatal("wrong result")
		}
	})
}

func TestQuirkSortIPAddrs(t *testing.T) {
	t.Run("with some addrs", func(t *testing.T) {
		addrs := []string{
			"::1",
			"192.168.1.2",
			"2a00:1450:4002:404::2004",
			"142.250.184.36",
			"2604:8800:5000:82:466:38ff:fecb:d46e",
			"198.145.29.83",
			"95.216.163.36",
		}
		expected := []string{
			"192.168.1.2",
			"142.250.184.36",
			"198.145.29.83",
			"95.216.163.36",
			"::1",
			"2a00:1450:4002:404::2004",
			"2604:8800:5000:82:466:38ff:fecb:d46e",
		}
		out := quirkSortIPAddrs(addrs)
		if diff := cmp.Diff(expected, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("with an empty list", func(t *testing.T) {
		if quirkSortIPAddrs(nil) != nil {
			t.Fatal("expected nil output")
		}
	})
}
