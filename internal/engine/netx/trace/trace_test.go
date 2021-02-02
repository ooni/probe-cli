package trace_test

import (
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

func TestGood(t *testing.T) {
	saver := trace.Saver{}
	var wg sync.WaitGroup
	const parallel = 10
	wg.Add(parallel)
	for idx := 0; idx < parallel; idx++ {
		go func() {
			saver.Write(trace.Event{})
			wg.Done()
		}()
	}
	wg.Wait()
	ev := saver.Read()
	if len(ev) != parallel {
		t.Fatal("unexpected number of events read")
	}
}
