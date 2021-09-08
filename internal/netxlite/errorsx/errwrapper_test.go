package errorsx

import (
	"encoding/json"
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
