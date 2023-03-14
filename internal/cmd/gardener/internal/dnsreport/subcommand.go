package dnsreport

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
)

// Subcommand is the dnsreport subcommand. The zero value is invalid; please, make
// sure you initialize all the fields marked as MANDATORY.
type Subcommand struct {
	// DNSOverHTTPSServerURL is the MANDATORY DoH server URL.
	DNSOverHTTPSServerURL string

	// JSONLOutputFile is the MANDATORY file where to write the results.
	JSONLOutputFile string

	// RepositoryDir is the MANDATORY directory where we cloned the test lists repository.
	RepositoryDir string
}

// Main is the main function of the dnsreport subcommand.
func (sc *Subcommand) Main(ctx context.Context) {
	// create wait groups for waiting background goroutines to join.
	readers := &sync.WaitGroup{}
	measurers := &sync.WaitGroup{}
	writers := &sync.WaitGroup{}

	// create channel for reading test list inputs
	inputs := make(chan *testlists.Entry)

	// read all the test lists entries in the background
	listsDir := filepath.Join(sc.RepositoryDir, "lists")
	readers.Add(1)
	go testlists.Generator(ctx, readers, listsDir, inputs)

	// create channel for reading the results
	outputs := make(chan *Measurement)

	// create workers for performing the measurement
	const workers = 8
	for idx := 0; idx < workers; idx++ {
		measurers.Add(1)
		go measurerWorker(ctx, measurers, idx, sc.DNSOverHTTPSServerURL, inputs, outputs)
	}

	// start working for writing the report on the disk
	writers.Add(1)
	go collectorWorker(ctx, writers, sc.JSONLOutputFile, outputs)

	// await for the readers and measures to finish their job
	readers.Wait()
	measurers.Wait()

	// signal the writer that we're done
	close(outputs)

	// await for the writers to finish their job
	writers.Wait()
}
