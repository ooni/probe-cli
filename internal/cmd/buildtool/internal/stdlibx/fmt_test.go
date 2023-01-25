package stdlibx

import (
	"bytes"
	"testing"
)

func TestMustFprintf(t *testing.T) {
	stdlib := NewStdlib()
	w := &bytes.Buffer{}
	stdlib.MustFprintf(w, "hello, %s\n", "world")
	if result := w.String(); result != "hello, world\n" {
		t.Fatal("unexpected result", result)
	}
}
