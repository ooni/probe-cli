package atomicx_test

import (
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestInt64(t *testing.T) {
	// TODO(bassosimone): how to write tests with race conditions
	// and be confident that they're WAI? Here I hope this test is
	// run with `-race` and I'm doing something that AFAICT will
	// be flagged as race if we were not be using mutexes.
	v := &atomicx.Int64{}
	go func() {
		v.Add(17)
	}()
	go func() {
		v.Add(14)
	}()
	time.Sleep(1 * time.Second)
	if v.Add(3) != 34 {
		t.Fatal("unexpected result")
	}
	if v.Load() != 34 {
		t.Fatal("unexpected result")
	}
}
