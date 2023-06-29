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

		// We do not set any address because we cannot be sure about the address, but the
		// trace is designed to operate on a single network connection, hence we do not need
		// to worry about multiple connections being involved. We COULD potentially have
		// more than a single destination with the [net.PacketConn] we're using for HTTP/3,
		// because in principle someone could send us lots of spurious packets that are
		// not meant for the QUIC connection while we're downloading, however this attack
		// seems quite unlikely in practice, so I think it's reasonable to conclude that
		// what the trace has seen is what the only conn in the trace has seen.
		if ev.Address != "" {
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
		if ev.Operation != throttling.BytesReceivedCumulativeOperation {
			t.Fatal("invalid operation", ev.Operation)
		}

		// We don't know the protocol. Again, this is not a problem because the trace is
		// designed to host a single connection and we have the transaction ID, which
		// mirrors the trace ID and tells us this information.
		if ev.Proto != "" {
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

		// This is important: we need to make sure the event's transaction ID mirrors the
		// trace ID, which is what allows us to attribte the performance events to the
		// specific connection we have created within the same trace ID.
		if ev.TransactionID != traceID {
			t.Fatal("unexpected transaction ID", ev.TransactionID, traceID)
		}

		// Make sure the tags are the ones we expect to see
		if diff := cmp.Diff(expectedTags, ev.Tags); diff != "" {
			t.Fatal(diff)
		}
	}
}
