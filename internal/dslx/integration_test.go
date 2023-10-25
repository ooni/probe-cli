package dslx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

func TestMakeSureWeCollectSpeedSamples(t *testing.T) {
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

	// instantiate a runtime
	rt := NewRuntimeMeasurexLite(model.DiscardLogger, time.Now())
	defer rt.Close()

	// create a measuring function
	f0 := Compose3(
		TCPConnect(rt),
		HTTPConnectionTCP(rt),
		HTTPRequest(rt),
	)

	// create the endpoint to measure
	epnt := &Endpoint{
		Address: server.Listener.Addr().String(),
		Domain:  "",
		Network: "tcp",
		Tags:    []string{},
	}

	// measure the endpoint
	result := f0.Apply(context.Background(), NewMaybeWithValue(epnt))

	// get observations
	observations := ExtractObservations(result)

	// process the network events and check for summary
	var foundSummary bool
	for _, entry := range observations {
		for _, ev := range entry.NetworkEvents {
			if ev.Operation == throttling.BytesReceivedCumulativeOperation {
				t.Log(ev)
				foundSummary = true
			}
		}
	}
	if !foundSummary {
		t.Fatal("did not find the summary")
	}
}
