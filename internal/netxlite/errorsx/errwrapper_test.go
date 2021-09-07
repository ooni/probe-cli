package errorsx

import (
	"errors"
	"io"
	"testing"
)

func TestErrWrapperError(t *testing.T) {
	err := &ErrWrapper{Failure: FailureDNSNXDOMAINError}
	if err.Error() != FailureDNSNXDOMAINError {
		t.Fatal("invalid return value")
	}
}

func TestErrWrapperUnwrap(t *testing.T) {
	err := &ErrWrapper{
		Failure:    FailureEOFError,
		WrappedErr: io.EOF,
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("cannot unwrap error")
	}
}
