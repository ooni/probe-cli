package dnsreport

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/aggregationapi"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/schollz/progressbar/v3"
)

// Subcommand is the dnsreport subcommand. The zero value is invalid; please, make
// sure you initialize all the fields marked as MANDATORY.
type Subcommand struct {
	// APIURL is the MANDATORY OONI API URL to use.
	APIURL string

	// CSVSummaryFile is the MANDATORY file where to write the CSV
	// summary file containing information on each failing URL.
	CSVSummaryFile string

	// DNSOverHTTPSServerURL is the MANDATORY DoH server URL.
	DNSOverHTTPSServerURL string

	// Force OPTIONALLY forces generating the JSONLOutputFile again.
	Force bool

	// JSONLCacheFile is the MANDATORY file where to cache the result of
	// performing a DNS measurement for each entry.
	JSONLCacheFile string

	// RepositoryDir is the MANDATORY directory where we previously
	// cloned the test lists repository.
	RepositoryDir string
}

// Main is the main function of the dnsreport subcommand.
func (sc *Subcommand) Main(ctx context.Context) {
	// generate the underlying cache if needed or if requested by the user
	if !fsx.RegularFileExists(sc.JSONLCacheFile) || sc.Force {
		sc.generateCache(ctx)
	}

	// now analyze the current cache file
	sc.analyzeCache(ctx)
}

// generateCache generates the JSONLCacheFile file by attempting to measure
// each URL in the test list using the DNSOverHTTPSServerURL.
func (sc *Subcommand) generateCache(ctx context.Context) {
	log.Infof("generating cache file: %s", sc.JSONLCacheFile)

	// create wait groups for waiting background goroutines to join.
	readers := &sync.WaitGroup{}
	measurers := &sync.WaitGroup{}
	writers := &sync.WaitGroup{}

	// create channel for reading test list inputs
	inputs := make(chan *testlists.Entry)

	// read all the test lists entries in the background
	listsDir := filepath.Join(sc.RepositoryDir, "lists")
	readers.Add(1)
	go testlists.Generator(readers, listsDir, inputs)

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
	go collectorWorker(ctx, writers, sc.JSONLCacheFile, outputs)

	// await for the readers and measures to finish their job
	readers.Wait()
	measurers.Wait()

	// signal the writer that we're done
	close(outputs)

	// await for the writers to finish their job
	writers.Wait()
}

// analyzeCache analyzes the content of the cache file.
func (sc *Subcommand) analyzeCache(ctx context.Context) {
	log.Infof("writing analysis results to %s", sc.CSVSummaryFile)

	// collect all cache entries
	all := sc.collectCacheEntries()

	// open the summary file
	filep := runtimex.Try1(os.Create(sc.CSVSummaryFile))

	// wrap the filep with a CSV writer
	writer := csv.NewWriter(filep)

	// write the first entry containing headers
	runtimex.Try0(writer.Write([]string{
		"file",
		"line",
		"url",
		"failure",
		"measurement_count",
		"anomaly_count",
		"confirmed_count",
		"ok_count",
		"failure_count",
	}))

	// create the progress bar to show the user progress
	bar := progressbar.NewOptions64(
		int64(len(all)),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetDescription(sc.JSONLCacheFile),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSetWriter(os.Stdout),
	)

	// process all the failed measurements
	for _, measurement := range all {
		bar.Add(1)
		if measurement.Failure != nil {
			apiResp := aggregationapi.Query(ctx, sc.APIURL, measurement.Entry.URL)
			runtimex.Try0(writer.Write([]string{
				measurement.Entry.File,
				strconv.Itoa(measurement.Entry.Line),
				measurement.Entry.URL,
				*measurement.Failure,
				strconv.FormatInt(apiResp.Result.MeasurementCount, 10),
				strconv.FormatInt(apiResp.Result.AnomalyCount, 10),
				strconv.FormatInt(apiResp.Result.ConfirmedCount, 10),
				strconv.FormatInt(apiResp.Result.OKCount, 10),
				strconv.FormatInt(apiResp.Result.FailureCount, 10),
			}))
		}
	}

	// make sure we successfully wrote all data
	writer.Flush()
	runtimex.Try0(writer.Error())
	runtimex.Try0(filep.Close())
}

// collectCacheEntries collects all the cache entries into a single array.
func (sc *Subcommand) collectCacheEntries() (out []*Measurement) {
	// open the cache file
	filep := runtimex.Try1(os.Open(sc.JSONLCacheFile))

	// walk through the lines
	scanner := bufio.NewScanner(filep)
	for scanner.Scan() {
		// parse the current line as a JSON
		data := scanner.Bytes()
		var measurement *Measurement
		runtimex.Try0(json.Unmarshal(data, &measurement))

		// append to output
		out = append(out, measurement)
	}

	// make sure we can close the scanner successfully
	runtimex.Try0(scanner.Err())

	// return collected entries to the caller
	return
}
