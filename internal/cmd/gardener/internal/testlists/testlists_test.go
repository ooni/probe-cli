package testlists_test

import (
	"context"
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
)

func TestWorkingAsIntended(t *testing.T) {
	// create controlling variables for the testlists.Generator
	ctx := context.Background()
	wg := &sync.WaitGroup{}
	och := make(chan *testlists.Entry)

	// run the generator in a background goroutine
	wg.Add(1)
	go testlists.Generator(ctx, wg, "testdata", och)

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

func TestInterrupted(t *testing.T) {
	// create controlling variables for the testlists.Generator
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}
	och := make(chan *testlists.Entry)

	// run the generator in a background goroutine
	wg.Add(1)
	go testlists.Generator(ctx, wg, "testdata", och)

	// collect all the generated entries
	var all []*testlists.Entry
	for entry := range och {
		all = append(all, entry)
		if len(all) > 15 {
			cancel()
			break
		}
	}

	// wait for the generator to terminate
	wg.Wait()

	if len(all) != 16 {
		t.Fatal("expected 16, got", len(all))
	}
}
