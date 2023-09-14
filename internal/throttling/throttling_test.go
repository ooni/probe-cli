package throttling

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestSamplerWorkingAsIntended(t *testing.T) {
	const (
		chunkSize   = 1 << 14
		repetitions = 10
		totalBody   = repetitions * chunkSize
		traceID     = 14
	)

	// create a testing server that sleeps after each send for a given number of sends
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := []byte(randx.Letters(chunkSize))
		for idx := 0; idx < repetitions; idx++ {
			w.Write(chunk)
			time.Sleep(250 * time.Millisecond)
		}
	}))
	defer server.Close()

	// create a trace
	expectedTags := []string{"antani", "mascetti"}
	tx := measurexlite.NewTrace(traceID, time.Now(), expectedTags...)

	// create a sampler for the trace
	sampler := NewSampler(tx)
	defer sampler.Close()

	// create a dialer
	dialer := tx.NewDialerWithoutResolver(model.DiscardLogger)

	// create an HTTP transport
	txp := netxlite.NewHTTPTransportLegacy(model.DiscardLogger, dialer, netxlite.NewNullTLSDialer())

	// create the HTTP request to issue
	req := runtimex.Try1(http.NewRequest("GET", server.URL, nil))

	// issue the HTTP request and await for response
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Log("got response", resp)

	// read the response body
	body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// make sure we've read the body
	if len(body) != totalBody {
		t.Fatal("expected", totalBody, "bytes but got", len(body), "bytes")
	}

	// make sure we have events to process
	events := sampler.ExtractSamples()
	if len(events) <= 0 {
		t.Fatal("expected to see at least one event")
	}

	// make sure each event looks good
	var (
		previousCounter int64
		previousT       float64
	)
	for _, ev := range events {
		t.Log(ev)

		// Make sure the address is the remote server address.
		if ev.Address != server.Listener.Addr().String() {
			t.Fatal("invalid address", ev.Address)
		}

		// There is no failure for this kind of events because we only collect statistics.
		if ev.Failure != nil {
			t.Fatal("invalid failure", ev.Failure)
		}

		// The number of bytes received should increase monotonically
		if ev.NumBytes < previousCounter {
			t.Fatal("non-monotonic bytes increase", ev.NumBytes, previousCounter)
		}
		previousCounter = ev.NumBytes

		// The operation should always be the expected one
		if ev.Operation != BytesReceivedCumulativeOperation {
			t.Fatal("invalid operation", ev.Operation)
		}

		// Make sure the protocol is the expected one
		if ev.Proto != "tcp" {
			t.Fatal("invalid proto", ev.Proto)
		}

		// The time should also increase monotonically. It may be possible for this test
		// to sometimes fail in cloud environments, based on other tests we have seen failing.
		if ev.T != ev.T0 {
			t.Fatal("T and T0 should be equal", ev.T, ev.T0)
		}
		if ev.T < previousT {
			t.Fatal("non-monotonic time increase", ev.T, previousT)
		}
		previousT = ev.T

		// Make sure the trace ID is the expected one
		if ev.TransactionID != traceID {
			t.Fatal("unexpected transaction ID", ev.TransactionID, traceID)
		}

		// Make sure the tags are the ones we expect to see
		if diff := cmp.Diff(expectedTags, ev.Tags); diff != "" {
			t.Fatal(diff)
		}
	}
}

func TestSampleSkipsInvalidMapEntries(t *testing.T) {
	// create a trace and a sampler
	tx := measurexlite.NewTrace(0, time.Now())
	sampler := NewSampler(tx)

	// create a fake map with an invalid entry and submit it
	stats := map[string]int64{
		"1.1.1.1:443":     128, // this entry is INVALID because it's missing the protocol
		"1.1.1.1:443/tcp": 44,  // INVALID because there's no space separator
	}

	// update the stats
	sampler.collectSnapshot(stats)

	// obtain the network events
	ev := sampler.ExtractSamples()
	if len(ev) != 0 {
		t.Fatal("expected to see no events here")
	}
}
