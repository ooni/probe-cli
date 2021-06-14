package atomicx_test

import (
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestInt64(t *testing.T) {
	v := &atomicx.Int64{}
	var wg sync.WaitGroup
	// many goroutines update the value in parallel
	for i := 0; i < 31; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v.Add(1)
		}()
	}
	wg.Wait()
	if v.Add(3) != 34 {
		t.Fatal("unexpected result")
	}
	if v.Load() != 34 {
		t.Fatal("unexpected result")
	}
}
