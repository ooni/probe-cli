package libminiooni_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/libminiooni"
)

func TestSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	libminiooni.MainWithConfiguration("example", libminiooni.Options{
		Yes: true,
	})
}
