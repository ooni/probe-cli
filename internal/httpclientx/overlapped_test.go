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

	//
	// Scenario:
	//
	// - 0.th.ooni.org is SNI blocked
	// - 1.th.ooni.org is SNI blocked
	// - 2.th.ooni.org is SNI blocked
	// - 3.th.ooni.org WAIs
	//
	// We expect to get a response from 3.th.ooni.org.
	//
	// Because the first three THs fail fast but the schedule interval is the default (i.e.,
	// 15 seconds), we're testing whether the algorithm allows us to recover quickly from
	// failure and check the other endpoints without waiting for too much time.
	//
	// Note: before changing the algorith,, this test ran for 45 seconds. Now it runs for 1s.
	//

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

	//
	// Scenario:
	//
	// - 0.th.ooni.org is WAI but slow
	// - 1.th.ooni.org is WAI but slow
	// - 2.th.ooni.org is WAI but slow
	// - 3.th.ooni.org is WAI but slow
	//
	// We expect to get a response from the first TH because it's the first goroutine
	// that we schedule and, even if the wakeup signals for THs are random, the schedule
	// interval is 15 seconds while we emit a wakeup signal every 0.25 seconds.
	//

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

	// we do not expect to see a failure because all the THs are WAI
	if err != nil {
		t.Fatal(err)
	}

	// compare response to expectation
	if diff := cmp.Diff(expectedResponse, apiResp); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewOverlappedPostJSONHandlesAllTimeouts(t *testing.T) {

	//
	// Scenario:
	//
	// - 0.th.ooni.org causes timeout
	// - 1.th.ooni.org causes timeout
	// - 2.th.ooni.org causes timeout
	// - 3.th.ooni.org causes timeout
	//
	// We expect to loop for all endpoints and then discover that all of them
	// failed. To make the test ~quick, we reduce the scheduling interval.
	//

	blockforever := make(chan any)

	zeroTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockforever
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer zeroTh.Close()

	oneTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockforever
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer oneTh.Close()

	twoTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockforever
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer twoTh.Close()

	threeTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockforever
		w.WriteHeader(http.StatusBadGateway)
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

	// make sure the schedule interval is low to make this test run faster.
	overlapped.ScheduleInterval = 250 * time.Millisecond

	// Now we issue the requests and check we're getting the correct response.
	//
	// IMPORTANT: here we need a context with timeout to ensure that we
	// eventually stop trying with the blocked-forever servers. In a more
	// real scenario, even without a context timeout, we have other
	// safeguards to unblock stuck readers in netxlite code.

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	apiResp, err := overlapped.Run(
		ctx,
		NewEndpoint(zeroTh.URL),
		NewEndpoint(oneTh.URL),
		NewEndpoint(twoTh.URL),
		NewEndpoint(threeTh.URL),
	)

	// we do not expect to see a failure because all the THs are WAI
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected error", err)
	}

	// we expect the api response to be nil
	if apiResp != nil {
		t.Fatal("expected non-nil resp")
	}

	// now unblock the blocked goroutines
	close(blockforever)
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
