package stdlibx

import (
	"errors"
	"testing"
)

func TestExitOnError(t *testing.T) {
	t.Run("without any error", func(t *testing.T) {
		stdlib := NewStdlib()
		stdlib.ExitOnError(nil, "foobar")
	})

	t.Run("without an error", func(t *testing.T) {
		expected := errors.New("mocked error")
		var got error
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = r.(error)
				}
			}()
			stdlib := NewStdlib().(*stdlib)
			stdlib.exiter = &testExiter{
				err: expected,
			}
			stdlib.ExitOnError(errors.New("antani"), "mascetti")
		}()
		if expected != got {
			t.Fatal("did not call exit")
		}
	})
}
