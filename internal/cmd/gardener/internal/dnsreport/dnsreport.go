// Package dnsreport implements the dnsreport subcommand.
package dnsreport

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/apex/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/aggregationapi"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/testlists"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/schollz/progressbar/v3"
)

// Subcommand is the dnsreport subcommand. The zero value is invalid; please, make
// sure you initialize all the fields marked as MANDATORY.
type Subcommand struct {
	// APIURL is the MANDATORY OONI API URL to use.
	APIURL string

	// DNSOverHTTPSServerURL is the MANDATORY DNS-over-HTTPS server URL.
	DNSOverHTTPSServerURL string

	// Database is the MANDATORY path of the database where to
	// store interim state while processing URLs.
	Database string

	// ReportFile is the MANDATORY file where to write the final report.
	ReportFile string

	// RepositoryDir is the MANDATORY directory where we previously
	// cloned the citizenlab/test-lists repository.
	RepositoryDir string
}

// Main is the main function of the dnsreport subcommand. This function calls
// [runtimex.PanicOnError] in case of failure.
func (s *Subcommand) Main(ctx context.Context) {
	// check whether the database exists
	dbExists := fsx.RegularFileExists(s.Database)

	// create or open the underlying sqlite3 database
	db := s.createOrOpenDatabase()
	defer db.Close()

	// if the database has just been created, then import
	// the URLs from the locally cloned git repository, otherwise
	// keep using the existing database file
	if !dbExists {
		log.Infof("creating new %s database", s.Database)
		s.loadFromRepository(db)
	} else {
		log.Infof("using existing %s database", s.Database)
	}

	// obtain the list of entries to measure
	entries := s.getEntriesToMeasure(db)
	log.Infof("we need to measure %d entries", len(entries))

	// measure each entry and update the database
	s.measureEntries(ctx, db, entries)

	// generate CSV report
	s.writeReport(db)
}

// createTableQuery is the query to create the dnsreport table.
const createTableQuery = `
CREATE TABLE IF NOT EXISTS dnsreport(
	file TEXT NOT NULL,
	line INTEGER NOT NULL,
	url TEXT NOT NULL,
	status TEXT NOT NULL,
	addresses TEXT NOT NULL,
	failure TEXT,
	measurement_count INTEGER NOT NULL,
	anomaly_count INTEGER NOT NULL,
	confirmed_count INTEGER NOT NULL,
	ok_count INTEGER NOT NULL,
	failure_count INTEGER NOT NULL
);
`

// createOrOpenDatabase is the function that either creates or
// opens the interim database containing status.
func (s *Subcommand) createOrOpenDatabase() *sql.DB {
	db := runtimex.Try1(sql.Open("sqlite3", s.Database))
	_ = runtimex.Try1(db.Exec(createTableQuery))
	return db
}

// insertIntoQuery is the query we use to insert a URL into the dnsreport table.
const insertIntoQuery = `
INSERT INTO dnsreport VALUES(
	?,
	?,
	?,
	?,
	'[]',
	NULL,
	0,
	0,
	0,
	0,
	0
)
`

// loadFromRepository loads URLs from the local repository clone
func (s *Subcommand) loadFromRepository(db *sql.DB) {
	log.Info("loading information from the github.com/citizenlab/test-lists repository")

	// create channel where to read the test list URLs
	och := make(chan *testlists.Entry)

	// create wait group to await for background goroutine to terminate
	wg := &sync.WaitGroup{}

	// start background worker goroutine
	wg.Add(1)
	go testlists.Generator(wg, filepath.Join(s.RepositoryDir, "lists"), och)

	// create transaction for inserting into the database
	tx := runtimex.Try1(db.Begin())
	defer tx.Commit()

	// read each entry and insert into transaction
	for entry := range och {
		_ = runtimex.Try1(tx.Exec(
			insertIntoQuery,
			entry.File,
			entry.Line,
			entry.URL,
			"inserted",
		))
	}
}

// selectInsertedQuery selects the entries to measure by checking the status
const selectInsertedQuery = `
SELECT rowid, file, line, url
FROM dnsreport
WHERE status = 'inserted';
`

// entryToMeasure contains data about an entry to measure.
type entryToMeasure struct {
	rowid int64
	file  string
	line  int64
	url   string
}

// getEntriesToMeasure gets the entries to measure from the database.
func (s *Subcommand) getEntriesToMeasure(db *sql.DB) (out []*entryToMeasure) {
	// execute the query and get the matching rows
	rows := runtimex.Try1(db.Query(selectInsertedQuery))
	defer rows.Close()

	// convert the rows to a list of [entryToMeasure]
	for rows.Next() {
		entry := &entryToMeasure{}
		runtimex.Try0(rows.Scan(&entry.rowid, &entry.file, &entry.line, &entry.url))
		out = append(out, entry)
	}

	// make sure there was no error while reading
	runtimex.Try0(rows.Err())

	// return list to the caller
	return
}

// measureEntries measures all the entries we need to measure
func (s *Subcommand) measureEntries(ctx context.Context, db *sql.DB, entries []*entryToMeasure) {
	// create the progress bar to show the user progress
	bar := progressbar.NewOptions64(
		int64(len(entries)),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stdout, "\n")
		}),
		progressbar.OptionSetWriter(os.Stdout),
	)

	// walk through each entry until we're interrupted by the context
	for idx := 0; idx < len(entries) && ctx.Err() == nil; idx++ {
		bar.Add(1)
		s.measureSingleEntry(db, entries[idx])
	}
}

// measureSingleEntry measures a single entry
func (s *Subcommand) measureSingleEntry(db *sql.DB, entry *entryToMeasure) {
	// parse the entry URL
	URL := runtimex.Try1(url.Parse(entry.url))
	hostname := URL.Hostname()

	// handle the input URLs where the domain is an IP address.
	if net.ParseIP(hostname) != nil {
		s.updateEntry(db, "skipped", []string{}, nil, nil, entry.rowid)
		return
	}

	// obtain the raw response and the error that occurs when trying
	// to obtain the result of LookupHost from the raw response
	addrs, err := s.dnsLookupANY(hostname)

	// if there is no error, stop processing right now and record
	// that we have measured the URL into the database.
	if err == nil {
		s.updateEntry(db, "ok", addrs, nil, nil, entry.rowid)
		return
	}

	// query the OONI API to obtain recent aggregate information about
	// measurement results for this URL
	apiResp := s.queryAggregationAPI(entry.url)

	// update the database
	s.updateEntry(db, "failed", []string{}, err, apiResp, entry.rowid)
}

// dnsLookupANY performs an ANY DNS lookup for the given domain. This function calls
// [runtimex.PanicOnError] for any network error and _only_ returns the error that
// arises from parsing the returned DNS response as a LookupHost response. The return
// value consists of (1) the raw response and (2) the error occurred when parsing
// the raw response as the result of a LookupHost query.
func (s *Subcommand) dnsLookupANY(domain string) ([]string, error) {
	// create countext bound to timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// create DNS transport using HTTP default client
	dnsTransport := netxlite.WrapDNSTransport(&netxlite.DNSOverHTTPSTransport{
		Client:       http.DefaultClient,
		Decoder:      &netxlite.DNSDecoderMiekg{},
		URL:          s.DNSOverHTTPSServerURL,
		HostOverride: "",
	})

	// create DNS resolver
	dnsResolver := netxlite.WrapResolver(
		log.Log,
		netxlite.NewUnwrappedParallelResolver(dnsTransport),
	)

	// lookup for both A and AAAA entries
	return dnsResolver.LookupHost(ctx, domain)
}

// queryAggregationAPI queries the aggregation API for the given URL.
func (s *Subcommand) queryAggregationAPI(inputURL string) *aggregationapi.Response {
	// create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// issue the query and return the response
	return aggregationapi.Query(ctx, s.APIURL, inputURL)
}

// updateQuery is the query to update a given entry.
const updateQuery = `
UPDATE dnsreport
SET status = ?,
    addresses = ?,
	failure = ?,
	measurement_count = ?,
	anomaly_count = ?,
	confirmed_count = ?,
	ok_count = ?,
	failure_count = ?
WHERE
	rowid = ?;
`

// updateEntry updates an entry into the database using
// the given DNS and OONI API results.
func (s *Subcommand) updateEntry(
	db *sql.DB,
	status string,
	addresses []string, // possibly nil
	err error, // possibly nil
	apiResp *aggregationapi.Response, // possibly nil
	rowid int64,
) {
	// create transaction for inserting into the database
	tx := runtimex.Try1(db.Begin())
	defer tx.Commit()

	// deal with apiResp possibly being nil
	var (
		measurementCount int64
		anomalyCount     int64
		confirmedCount   int64
		okCount          int64
		failureCount     int64
	)
	if apiResp != nil {
		measurementCount = apiResp.Result.MeasurementCount
		anomalyCount = apiResp.Result.AnomalyCount
		confirmedCount = apiResp.Result.ConfirmedCount
		okCount = apiResp.Result.OKCount
		failureCount = apiResp.Result.FailureCount
	}

	// update the existing row with new information
	_ = runtimex.Try1(tx.Exec(
		updateQuery,
		status,
		string(runtimex.Try1(json.Marshal(addresses))),
		measurexlite.NewFailure(err), // deals with nil gracefully
		measurementCount,
		anomalyCount,
		confirmedCount,
		okCount,
		failureCount,
		rowid,
	))
}

// selectFailedQuery selects the entries that have failed
const selectFailedQuery = `
SELECT file, line, url, failure, measurement_count,
	anomaly_count, confirmed_count, ok_count, failure_count
FROM dnsreport
WHERE status = 'failed';
`

// writeReport writes a CSV report containing the results inside the
// database that should be examined by researchers.
func (s *Subcommand) writeReport(db *sql.DB) {
	// logging
	log.Infof("writing researchers' report file: %s", s.ReportFile)

	// create the output file
	filep := runtimex.Try1(os.Create(s.ReportFile))

	// create the CSV writer wrapper
	writer := csv.NewWriter(filep)

	// write the first CSV row with headers
	runtimex.Try0(writer.Write([]string{
		"file", "line", "url", "failure", "measurement_count",
		"anomaly_count", "confirmed_count", "ok_count", "failure_count",
	}))
	writer.Flush()

	// query all the entries that have been measured
	rows := runtimex.Try1(db.Query(selectFailedQuery))
	defer rows.Close()

	// write each row into the CSV file
	for rows.Next() {
		// read from query
		var (
			file             string
			line             int64
			url              string
			failure          *string
			measurementCount int64
			anomalyCount     int64
			confirmedCount   int64
			okCount          int64
			failureCount     int64
		)
		runtimex.Try0(rows.Scan(
			&file, &line, &url, &failure, &measurementCount, &anomalyCount,
			&confirmedCount, &okCount, &failureCount,
		))

		// sanity check with respect to the failed state
		runtimex.Assert(failure != nil, "expected non-nil failure")

		// write to CSV
		runtimex.Try0(writer.Write([]string{
			file,
			strconv.FormatInt(line, 10),
			url,
			*failure,
			strconv.FormatInt(measurementCount, 10),
			strconv.FormatInt(anomalyCount, 10),
			strconv.FormatInt(confirmedCount, 10),
			strconv.FormatInt(okCount, 10),
			strconv.FormatInt(failureCount, 10),
		}))
		writer.Flush()
	}

	// make sure there was no error while reading
	runtimex.Try0(rows.Err())

	// make sure there was no error while writing
	runtimex.Try0(writer.Error())
	runtimex.Try0(filep.Close())
}
