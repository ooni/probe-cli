package testlists_test

import (
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
)

func TestWorkingAsIntended(t *testing.T) {
	// create controlling variables for the testlists.Generator
	wg := &sync.WaitGroup{}
	och := make(chan *testlists.Entry)

	// run the generator in a background goroutine
	wg.Add(1)
	go testlists.Generator(wg, "testdata", och)

	// collect all the generated entries
	var all []*testlists.Entry
	for entry := range och {
		all = append(all, entry)
	}

	// wait for the generator to terminate
	wg.Wait()

	if len(all) != 1860 {
		t.Fatal("expected 1860, got", len(all))
	}
}
