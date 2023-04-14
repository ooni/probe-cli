package dnsfix_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/dnsfix"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

func TestWorkingAsIntended(t *testing.T) {
	// copy the original CSV file so we modify a copy
	orig := filepath.Join("testdata", "lists", "it.csv")
	copied := filepath.Join("testdata", "lists", "it-copy.csv")
	if err := shellx.CopyFile(orig, copied, 0644); err != nil {
		t.Fatal(err)
	}

	// fix the test list according to the dnsreport.csv file
	subc := &dnsfix.Subcommand{
		ReportFile: filepath.Join("testdata", "dnsreport.csv"),
	}
	subc.Main()

	// make sure we get the expected changes
	expectFile := filepath.Join("testdata", "lists", "it-expected.csv")
	expect, err := os.ReadFile(expectFile)
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
