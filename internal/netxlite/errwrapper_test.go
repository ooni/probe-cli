package netxlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
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

	t.Run("MarshalJSON", func(t *testing.T) {
		wrappedErr := &ErrWrapper{
			Failure:    FailureEOFError,
			WrappedErr: io.EOF,
		}
		data, err := json.Marshal(wrappedErr)
		if err != nil {
			t.Fatal(err)
		}
		s := string(data)
		if s != "\""+FailureEOFError+"\"" {
			t.Fatal("invalid serialization", s)
		}
	})
}

func TestNewErrWrapper(t *testing.T) {
	t.Run("panics if the classifier is nil", func(t *testing.T) {
		recovered := &atomicx.Int64{}
		func() {
			defer func() {
				if recover() != nil {
					recovered.Add(1)
				}
			}()
			NewErrWrapper(nil, CloseOperation, io.EOF)
		}()
		if recovered.Load() != 1 {
			t.Fatal("did not panic")
		}
	})

	t.Run("panics if the operation is empty", func(t *testing.T) {
		recovered := &atomicx.Int64{}
		func() {
			defer func() {
				if recover() != nil {
					recovered.Add(1)
				}
			}()
			NewErrWrapper(ClassifyGenericError, "", io.EOF)
		}()
		if recovered.Load() != 1 {
			t.Fatal("did not panic")
		}
	})

	t.Run("panics if the error is nil", func(t *testing.T) {
		recovered := &atomicx.Int64{}
		func() {
			defer func() {
				if recover() != nil {
					recovered.Add(1)
				}
			}()
			NewErrWrapper(ClassifyGenericError, CloseOperation, nil)
		}()
		if recovered.Load() != 1 {
			t.Fatal("did not panic")
		}
	})

	t.Run("otherwise, works as intended", func(t *testing.T) {
		ew := NewErrWrapper(ClassifyGenericError, CloseOperation, io.EOF)
		if ew.Failure != FailureEOFError {
			t.Fatal("unexpected failure")
		}
		if ew.Operation != CloseOperation {
			t.Fatal("unexpected operation")
		}
		if ew.WrappedErr != io.EOF {
			t.Fatal("unexpected WrappedErr")
		}
	})

	t.Run("when the underlying error is already a wrapped error", func(t *testing.T) {
		ew := NewErrWrapper(classifySyscallError, ReadOperation, ECONNRESET)
		var err1 error = ew
		err2 := fmt.Errorf("cannot read: %w", err1)
		ew2 := NewErrWrapper(ClassifyGenericError, TopLevelOperation, err2)
		if ew2.Failure != ew.Failure {
			t.Fatal("not the same failure")
		}
		if ew2.Operation != ew.Operation {
			t.Fatal("not the same operation")
		}
		if ew2.WrappedErr != err2 {
			t.Fatal("invalid underlying error")
		}
	})
}

func TestNewTopLevelGenericErrWrapper(t *testing.T) {
	out := NewTopLevelGenericErrWrapper(io.EOF)
	if out.Failure != FailureEOFError {
		t.Fatal("invalid failure")
	}
	if out.Operation != TopLevelOperation {
		t.Fatal("invalid operation")
	}
	if !errors.Is(out, io.EOF) {
		t.Fatal("invalid underlying error using errors.Is")
	}
	if !errors.Is(out.WrappedErr, io.EOF) {
		t.Fatal("invalid WrappedErr")
	}
}
