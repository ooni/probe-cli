package throttling_test

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
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

func TestSampler(t *testing.T) {
	const (
		chunkSize   = 1 << 14
		repetitions = 10
		totalBody   = repetitions * chunkSize
		traceID     = 14
	)

	// create a testing server that sleeps after each sender for a given number of sends
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
	sampler := throttling.NewSampler(tx)
	defer sampler.Close()

	// create a dialer
	dialer := tx.NewDialerWithoutResolver(model.DiscardLogger)

	// create an HTTP transport
	txp := netxlite.NewHTTPTransport(model.DiscardLogger, dialer, netxlite.NewNullTLSDialer())

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

	events := sampler.ExtractSamples()
	var previousCounter int64
	var previousT float64
	for _, ev := range events {
		t.Log(ev)

		if ev.Address != "" {
			t.Fatal("invalid address", ev.Address)
		}

		if ev.Failure != nil {
			t.Fatal("invalid failure", ev.Failure)
		}

		if ev.NumBytes < previousCounter {
			t.Fatal("non-monotonic bytes increase", ev.NumBytes, previousCounter)
		}
		previousCounter = ev.NumBytes

		if ev.Operation != throttling.BytesReceivedCumulativeOperation {
			t.Fatal("invalid operation", ev.Operation)
		}

		if ev.Proto != "" {
			t.Fatal("invalid proto", ev.Proto)
		}

		if ev.T != ev.T0 {
			t.Fatal("T and T0 should be equal", ev.T, ev.T0)
		}
		if ev.T < previousT {
			t.Fatal("non-monotonic time increase", ev.T, previousT)
		}
		previousT = ev.T

		if ev.TransactionID != traceID {
			t.Fatal("unexpected transaction ID", ev.TransactionID, traceID)
		}

		if diff := cmp.Diff(expectedTags, ev.Tags); diff != "" {
			t.Fatal(diff)
		}
	}
}
