package errorsx

import (
	"errors"
	"io"
	"testing"
)

func TestErrWrapper(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		err := &ErrWrapper{Failure: FailureDNSNXDOMAINError}
		if err.Error() != FailureDNSNXDOMAINError {
			t.Fatal("invalid return value")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		err := &ErrWrapper{
			Failure:    FailureEOFError,
			WrappedErr: io.EOF,
		}
		if !errors.Is(err, io.EOF) {
			t.Fatal("cannot unwrap error")
		}
	})
}
