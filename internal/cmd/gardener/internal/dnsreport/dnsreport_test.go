package dnsreport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/aggregationapi"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// newServerDNSOverHTTPS creates a fake DNS-over-HTTPS server.
func newServerDNSOverHTTPS() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read incoming DNS query
		data, err := netxlite.ReadAllContext(r.Context(), r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}

		// parse incoming DNS query
		query := &dns.Msg{}
		if err := query.Unpack(data); err != nil {
			w.WriteHeader(400)
			return
		}

		// obtain the query question
		runtimex.Assert(len(query.Question) >= 1, "no questions")
		q0 := query.Question[0]

		// create DNS response
		resp := &dns.Msg{}
		if q0.Name == "torrentroom.com." {
			resp.SetReply(query)
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:     "torrentroom.com.",
					Rrtype:   dns.TypeA,
					Class:    dns.ClassINET,
					Ttl:      1234,
					Rdlength: 0,
				},
				A: net.IPv4(8, 9, 10, 11),
			})
		} else {
			resp.SetRcode(query, dns.RcodeNameError)
		}

		// serialize and return the reponse
		data, err = resp.Pack()
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Header().Add("content-type", "application/dns-message")
		w.Write(data)
	}))
}

// newServerOONIAPI creates a fake OONI API server.
func newServerOONIAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := &aggregationapi.Response{
			Result: aggregationapi.ResponseResult{
				AnomalyCount:     10,
				ConfirmedCount:   4,
				FailureCount:     7,
				MeasurementCount: 30,
				OKCount:          9,
			},
			V: 0,
		}
		data := runtimex.Try1(json.Marshal(resp))
		w.Write(data)
	}))
}

const selectQueryForTesting = `
SELECT file, line, url, status, addresses, failure,
	measurement_count, anomaly_count, confirmed_count,
	ok_count, failure_count
FROM dnsreport;
`

// validateDatabaseResults verifies that we have the correct results in the DB.
func validateDatabaseResults(dbPath string) error {
	// open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// issue the select query
	rows, err := db.Query(selectQueryForTesting)
	if err != nil {
		return err
	}

	// iterate over each result
	var count int64
	for rows.Next() {
		count++

		// read the row from the database
		var (
			file             string
			line             int64
			url              string
			status           string
			addresses        string
			failure          *string
			measurementCount int64
			anomalyCount     int64
			confirmedCount   int64
			okCount          int64
			failureCount     int64
		)
		err := rows.Scan(
			&file,
			&line,
			&url,
			&status,
			&addresses,
			&failure,
			&measurementCount,
			&anomalyCount,
			&confirmedCount,
			&okCount,
			&failureCount,
		)
		if err != nil {
			return err
		}

		// the file is common to all entries
		expectedFile := filepath.Join("testdata", "repo", "lists", "it.csv")
		if file != expectedFile {
			return fmt.Errorf("expected %s and got %s", expectedFile, file)
		}

		// construct a list with the counters from the API to ease
		// comparing them with expectations using cmp.Diff
		apiCounters := []int64{
			anomalyCount,
			confirmedCount,
			failureCount,
			measurementCount,
			okCount,
		}

		// check the results depending on the url
		switch url {
		case "http://www.torrentdownload.ws/":
			if line != 2 {
				return fmt.Errorf("expected line to be 2, got %d", line)
			}
			if status != "failed" {
				return fmt.Errorf("expected status to be failed, got %s", status)
			}
			if addresses != "[]" {
				return fmt.Errorf("expected addresses to be [], got %s", addresses)
			}
			if failure == nil {
				return errors.New("expected non-nil failure")
			}
			if *failure != "dns_nxdomain_error" {
				return fmt.Errorf("expected *failure to be dns_nxdomain_error, got %s", *failure)
			}
			expectCounters := []int64{10, 4, 7, 30, 9}
			if diff := cmp.Diff(expectCounters, apiCounters); diff != "" {
				return fmt.Errorf("unexpected API counters: %s", diff)
			}

		case "http://130.192.91.211/":
			if line != 3 {
				return fmt.Errorf("expected line to be 3, got %d", line)
			}
			if status != "skipped" {
				return fmt.Errorf("expected status to be skipped, got %s", status)
			}
			if addresses != "[]" {
				return fmt.Errorf("expected addresses to be [], got %s", addresses)
			}
			if failure != nil {
				return errors.New("expected nil failure")
			}
			expectCounters := []int64{0, 0, 0, 0, 0}
			if diff := cmp.Diff(expectCounters, apiCounters); diff != "" {
				return fmt.Errorf("unexpected API counters: %s", diff)
			}

		case "http://torrentroom.com/":
			if line != 4 {
				return fmt.Errorf("expected line to be 4, got %d", line)
			}
			if status != "ok" {
				return fmt.Errorf("expected status to be ok, got %s", status)
			}
			if addresses != `["8.9.10.11"]` {
				return fmt.Errorf("expected addresses to be [\"8.9.10.11\"], got %s", addresses)
			}
			if failure != nil {
				return errors.New("expected nil failure")
			}
			expectCounters := []int64{0, 0, 0, 0, 0}
			if diff := cmp.Diff(expectCounters, apiCounters); diff != "" {
				return fmt.Errorf("unexpected API counters: %s", diff)
			}

		default:
			return fmt.Errorf("got unexpected URL: %s", url)
		}
	}

	// make sure we had exactly three entries
	if count != 3 {
		return fmt.Errorf("expected 3 entries, got %d", count)
	}

	// make sure there was no error
	return rows.Err()
}

// validateResultsCSV validates the contents of the CSV file.
func validateResultsCSV(csvPath string) error {
	// read the CSV generated by the tests
	got, err := os.ReadFile(csvPath)
	if err != nil {
		return err
	}

	// read the expected CSV file
	expectedPath := filepath.Join("testdata", "dnsreport-expected.csv")
	expect, err := os.ReadFile(expectedPath)
	if err != nil {
		return err
	}

	// compare results to expectation
	if diff := cmp.Diff(expect, got); diff != "" {
		return fmt.Errorf("unexpected CSV file content: %s", diff)
	}

	return nil
}

func TestWorkingAsIntended(t *testing.T) {
	// create DNS-over-HTTPS server running on localhost
	dnsSrvr := newServerDNSOverHTTPS()
	defer dnsSrvr.Close()

	// create OONI API server running on localhost
	apiSrvr := newServerOONIAPI()
	defer apiSrvr.Close()

	// initialize the dnsreport subcommand
	databaseFile := filepath.Join("testdata", "dnsreport.sqlite3")
	repoDir := filepath.Join("testdata", "repo")
	reportFile := filepath.Join("testdata", "dnsreport.csv")
	sc := &Subcommand{
		APIURL:                apiSrvr.URL,
		DNSOverHTTPSServerURL: dnsSrvr.URL,
		Database:              databaseFile,
		ReportFile:            reportFile,
		RepositoryDir:         repoDir,
	}

	t.Run("without pre-existing database", func(t *testing.T) {
		// make sure there is no databaseFile when testing
		runtimex.Try0(os.RemoveAll(databaseFile))

		// run the main function of the subcommand
		sc.Main(context.Background())

		// validate the results
		if err := validateDatabaseResults(databaseFile); err != nil {
			t.Fatal(err)
		}
		if err := validateResultsCSV(reportFile); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with a pre-existing database", func(t *testing.T) {
		// make sure there is no databaseFile when testing
		runtimex.Try0(os.RemoveAll(databaseFile))

		// run the main function of the subcommand with a cancelled context, which
		// should prevent us from processing the URLs
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately
		sc.Main(ctx)

		// run again with background context, which should cause us to
		// start processing from a pre-existing database.
		sc.Main(context.Background())

		// validate the results
		if err := validateDatabaseResults(databaseFile); err != nil {
			t.Fatal(err)
		}
		if err := validateResultsCSV(reportFile); err != nil {
			t.Fatal(err)
		}
	})
}
