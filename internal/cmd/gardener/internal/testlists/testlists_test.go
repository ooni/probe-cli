package testlists_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

func TestGenerator(t *testing.T) {
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

	if len(all) != 28 {
		t.Fatal("expected 28, got", len(all))
	}
}

func TestRewrite(t *testing.T) {
	// create a copy of the test list we want to rewrite
	orig := filepath.Join("testdata", "it.csv")
	copied := filepath.Join("testdata", "it-copy.csv")
	if err := shellx.CopyFile(orig, copied, 0644); err != nil {
		t.Fatal(err)
	}

	// rewrite the test list keeping only the entries containing "torrent" in their name
	shouldKeep := func(URL string) bool {
		return strings.Contains(URL, "torrent")
	}
	testlists.Rewrite(copied, shouldKeep)

	// make sure the resulting file is what we expected
	expectedFile := filepath.Join("testdata", "it-expected.csv")
	expect, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(copied)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}
