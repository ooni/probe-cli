package httpclientx

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// Implementation note: because top-level functions such as GetRaw always use
// an [*Overlapped], we do not necessarily need to test that each top-level constructor
// are WAI; rather, we should focus on the mechanics of multiple URLs.

func TestNewOverlappedPostJSONIsPerformingOverlappedCalls(t *testing.T) {

	// Scenario:
	//
	// - 0.th.ooni.org is SNI blocked
	// - 1.th.ooni.org is SNI blocked
	// - 2.th.ooni.org is SNI blocked
	// - 3.th.ooni.org WAIs

	zeroTh := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
	defer zeroTh.Close()

	oneTh := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
	defer oneTh.Close()

	twoTh := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
	defer twoTh.Close()

	expectedResponse := &apiResponse{
		Age:  41,
		Name: "sbs",
	}

	threeTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer threeTh.Close()

	// Create client configuration. We don't care much about the
	// JSON requests and reponses being aligned to reality.

	apiReq := &apiRequest{
		UserID: 117,
	}

	overlapped := NewOverlappedPostJSON[*apiRequest, *apiResponse](apiReq, &Config{
		Authorization: "", // not relevant for this test
		Client:        http.DefaultClient,
		Logger:        log.Log,
		UserAgent:     model.HTTPHeaderUserAgent,
	})

	// make sure we set a low scheduling interval to make test faster
	overlapped.ScheduleInterval = time.Second

	// Now we issue the requests and check we're getting the correct response.

	apiResp, err := overlapped.Run(
		context.Background(),
		NewEndpoint(zeroTh.URL),
		NewEndpoint(oneTh.URL),
		NewEndpoint(twoTh.URL),
		NewEndpoint(threeTh.URL),
	)

	// we do not expect to see a failure because threeTh is WAI
	if err != nil {
		t.Fatal(err)
	}

	// compare response to expectation
	if diff := cmp.Diff(expectedResponse, apiResp); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewOverlappedPostJSONCancelsPendingCalls(t *testing.T) {

	// Scenario:
	//
	// - 0.th.ooni.org is WAI but slow
	// - 1.th.ooni.org is WAI
	// - 2.th.ooni.org is WAI
	// - 3.th.ooni.org is WAI

	expectedResponse := &apiResponse{
		Age:  41,
		Name: "sbs",
	}

	slowwakeup := make(chan any)

	zeroTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-slowwakeup
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer zeroTh.Close()

	oneTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-slowwakeup
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer oneTh.Close()

	twoTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-slowwakeup
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer twoTh.Close()

	threeTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-slowwakeup
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer threeTh.Close()

	// Create client configuration. We don't care much about the
	// JSON requests and reponses being aligned to reality.

	apiReq := &apiRequest{
		UserID: 117,
	}

	overlapped := NewOverlappedPostJSON[*apiRequest, *apiResponse](apiReq, &Config{
		Authorization: "", // not relevant for this test
		Client:        http.DefaultClient,
		Logger:        log.Log,
		UserAgent:     model.HTTPHeaderUserAgent,
	})

	// make sure the schedule interval is high because we want
	// all the goroutines but the first to be waiting for permission
	// to fetch from their respective URLs.
	overlapped.ScheduleInterval = 15 * time.Second

	// In the background we're going to emit slow wakeup signals at fixed intervals
	// after an initial waiting interval, such that goroutines unblock in order

	go func() {
		time.Sleep(250 * time.Millisecond)
		for idx := 0; idx < 4; idx++ {
			slowwakeup <- true
			time.Sleep(250 * time.Millisecond)
		}
		close(slowwakeup)
	}()

	// Now we issue the requests and check we're getting the correct response.

	apiResp, err := overlapped.Run(
		context.Background(),
		NewEndpoint(zeroTh.URL),
		NewEndpoint(oneTh.URL),
		NewEndpoint(twoTh.URL),
		NewEndpoint(threeTh.URL),
	)

	// we do not expect to see a failure because threeTh is WAI
	if err != nil {
		t.Fatal(err)
	}

	// compare response to expectation
	if diff := cmp.Diff(expectedResponse, apiResp); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewOverlappedPostJSONWithNoURLs(t *testing.T) {

	// Create client configuration. We don't care much about the
	// JSON requests and reponses being aligned to reality.

	apiReq := &apiRequest{
		UserID: 117,
	}

	overlapped := NewOverlappedPostJSON[*apiRequest, *apiResponse](apiReq, &Config{
		Authorization: "", // not relevant for this test
		Client:        http.DefaultClient,
		Logger:        log.Log,
		UserAgent:     model.HTTPHeaderUserAgent,
	})

	// Now we issue the requests without any URLs and make sure
	// the result we get is the generic overlapped error

	apiResp, err := overlapped.Run(context.Background())

	// we do not expect to see a failure because threeTh is WAI
	if !errors.Is(err, ErrGenericOverlappedFailure) {
		t.Fatal("unexpected error", err)
	}

	// we expect a nil response
	if apiResp != nil {
		t.Fatal("expected nil API response")
	}
}
