package dnsreport_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/dnsreport"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestWorkingAsIntended(t *testing.T) {
	// create DNS over HTTPS server running on localhost
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// create DNS response
		resp := &dns.Msg{}
		resp.SetRcode(query, dns.RcodeNameError)

		// serialize and return the reponse
		data, err = resp.Pack()
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Write(data)
	}))
	defer srvr.Close()

	// initialize and run the dnsreport subcommand
	summaryFile := filepath.Join("testdata", "dnsreport.csv")
	outputFile := filepath.Join("testdata", "dnsreport.jsonl")
	sc := &dnsreport.Subcommand{
		CSVSummaryFile:        summaryFile,
		DNSOverHTTPSServerURL: srvr.URL,
		Force:                 true,
		JSONLCacheFile:        outputFile,
		RepositoryDir:         filepath.Join("testdata", "repo"),
	}
	sc.Main(context.Background())

	// make sure we can open the output file
	filep, err := os.Open(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	// count the number of entries
	scanner := bufio.NewScanner(filep)
	var entries int
	for scanner.Scan() {
		data := scanner.Bytes()
		var measurement *dnsreport.Measurement
		if err := json.Unmarshal(data, &measurement); err != nil {
			t.Fatal(err)
		}
		entries++
	}

	// make sure we can close the scanner successfully
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	// make sure we have exactly two entries
	if entries != 2 {
		t.Fatal("expected 2 entries, got", entries)
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	// create DNS over HTTPS server running on localhost
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// create DNS response
		resp := &dns.Msg{}
		resp.SetRcode(query, dns.RcodeNameError)

		// serialize and return the reponse
		data, err = resp.Pack()
		if err != nil {
			w.WriteHeader(500)
			return
		}
		w.Write(data)
	}))
	defer srvr.Close()

	// create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately to test what happens when we cancel the context

	// initialize and run the dnsreport subcommand
	summaryFile := filepath.Join("testdata", "dnsreport-cancelled.csv")
	outputFile := filepath.Join("testdata", "dnsreport-cancelled.jsonl")
	sc := &dnsreport.Subcommand{
		CSVSummaryFile:        summaryFile,
		DNSOverHTTPSServerURL: srvr.URL,
		Force:                 true,
		JSONLCacheFile:        outputFile,
		RepositoryDir:         filepath.Join("testdata", "repo"),
	}
	sc.Main(ctx)

	// make sure we can open the output file
	filep, err := os.Open(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	// count the number of entries
	scanner := bufio.NewScanner(filep)
	var entries int
	for scanner.Scan() {
		data := scanner.Bytes()
		var measurement *dnsreport.Measurement
		if err := json.Unmarshal(data, &measurement); err != nil {
			t.Fatal(err)
		}
		entries++
	}

	// make sure we can close the scanner successfully
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	// make sure we have exactly two entries
	if entries != 0 {
		t.Fatal("expected 0 entries, got", entries)
	}
}
