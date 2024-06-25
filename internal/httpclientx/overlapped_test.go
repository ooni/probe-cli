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
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// Implementation note: because top-level functions such as GetRaw always use
// an [*Overlapped], we do not necessarily need to test that each top-level constructor
// are WAI; rather, we should focus on the mechanics of multiple URLs.

func TestNewOverlappedPostJSONFastRecoverFromEarlyErrors(t *testing.T) {

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
	// failure and check the other base URLs without waiting for too much time.
	//
	// Note: before changing the algorithm, this test ran for 45 seconds. Now it runs
	// for 1s because a previous goroutine terminating with error causes the next
	// goroutine to start and attempt to fetch the resource.
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
	//
	// We're splitting the algorithm into its Map step and its Reduce step because
	// this allows us to clearly observe what happened.

	results := overlapped.Map(
		context.Background(),
		NewBaseURL(zeroTh.URL),
		NewBaseURL(oneTh.URL),
		NewBaseURL(twoTh.URL),
		NewBaseURL(threeTh.URL),
	)

	runtimex.Assert(len(results) == 4, "unexpected number of results")

	// the first three attempts should have failed with connection reset
	// while the fourth result should be successful
	for _, entry := range results {
		t.Log(entry.Index, string(must.MarshalJSON(entry)))
		switch entry.Index {
		case 0, 1, 2:
			if err := entry.Err; !errors.Is(err, netxlite.ECONNRESET) {
				t.Fatal("unexpected error", err)
			}
		case 3:
			if err := entry.Err; err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(expectedResponse, entry.Value); diff != "" {
				t.Fatal(diff)
			}
		default:
			t.Fatal("unexpected index", entry.Index)
		}
	}

	// Now run the reduce step of the algorithm and make sure we correctly
	// return the first success and the nil error

	apiResp, idx, err := OverlappedReduce(results)

	// we do not expect to see a failure because threeTh is WAI
	if err != nil {
		t.Fatal(err)
	}

	if idx != 3 {
		t.Fatal("unexpected success index", idx)
	}

	// compare response to expectation
	if diff := cmp.Diff(expectedResponse, apiResp); diff != "" {
		t.Fatal(diff)
	}
}

func TestNewOverlappedPostJSONFirstCallSucceeds(t *testing.T) {

	//
	// Scenario:
	//
	// - 0.th.ooni.org is WAI
	// - 1.th.ooni.org is WAI
	// - 2.th.ooni.org is WAI
	// - 3.th.ooni.org is WAI
	//
	// We expect to get a response from the first TH because it's the first goroutine
	// that we schedule. Subsequent calls should be canceled.
	//

	expectedResponse := &apiResponse{
		Age:  41,
		Name: "sbs",
	}

	zeroTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer zeroTh.Close()

	oneTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer oneTh.Close()

	twoTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(must.MarshalJSON(expectedResponse))
	}))
	defer twoTh.Close()

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

	// make sure the schedule interval is high because we want
	// all the goroutines but the first to be waiting for permission
	// to fetch from their respective URLs.
	overlapped.ScheduleInterval = 15 * time.Second

	// Now we issue the requests and check we're getting the correct response.
	//
	// We're splitting the algorithm into its Map step and its Reduce step because
	// this allows us to clearly observe what happened.

	results := overlapped.Map(
		context.Background(),
		NewBaseURL(zeroTh.URL),
		NewBaseURL(oneTh.URL),
		NewBaseURL(twoTh.URL),
		NewBaseURL(threeTh.URL),
	)

	runtimex.Assert(len(results) == 4, "unexpected number of results")

	// the first attempt should succeed and subsequent ones should
	// have failed with the context.Canceled error
	for _, entry := range results {
		t.Log(entry.Index, string(must.MarshalJSON(entry)))
		switch entry.Index {
		case 1, 2, 3:
			if err := entry.Err; !errors.Is(err, context.Canceled) {
				t.Fatal("unexpected error", err)
			}
		case 0:
			if err := entry.Err; err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(expectedResponse, entry.Value); diff != "" {
				t.Fatal(diff)
			}
		default:
			t.Fatal("unexpected index", entry.Index)
		}
	}

	// Now run the reduce step of the algorithm and make sure we correctly
	// return the first success and the nil error

	apiResp, idx, err := OverlappedReduce(results)

	// we do not expect to see a failure because all the THs are WAI
	if err != nil {
		t.Fatal(err)
	}

	if idx != 0 {
		t.Fatal("unexpected success index", idx)
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
	// We expect to loop for all base URLs and then discover that all of them
	// failed. To make the test ~quick, we reduce the scheduling interval, and
	// the watchdog timeout.
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
	// We're splitting the algorithm into its Map step and its Reduce step because
	// this allows us to clearly observe what happened.

	// modify the watchdog timeout be much smaller than usual
	overlapped.WatchdogTimeout = 2 * time.Second

	results := overlapped.Map(
		context.Background(),
		NewBaseURL(zeroTh.URL),
		NewBaseURL(oneTh.URL),
		NewBaseURL(twoTh.URL),
		NewBaseURL(threeTh.URL),
	)

	runtimex.Assert(len(results) == 4, "unexpected number of results")

	// all the attempts should have failed with context deadline exceeded
	for _, entry := range results {
		t.Log(entry.Index, string(must.MarshalJSON(entry)))
		switch entry.Index {
		case 0, 1, 2, 3:
			if err := entry.Err; !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("unexpected error", err)
			}
		default:
			t.Fatal("unexpected index", entry.Index)
		}
	}

	// Now run the reduce step of the algorithm and make sure we correctly
	// return the first success and the nil error

	apiResp, idx, err := OverlappedReduce(results)

	// we expect to see a failure because the watchdog timeout should have fired
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected error", err)
	}

	if idx != 0 {
		t.Fatal("unexpected index", idx)
	}

	// we expect the api response to be nil
	if apiResp != nil {
		t.Fatal("expected nil resp")
	}

	// now unblock the blocked goroutines
	close(blockforever)
}

func TestNewOverlappedPostJSONResetTimeoutSuccessCanceled(t *testing.T) {

	//
	// Scenario:
	//
	// - 0.th.ooni.org resets the connection
	// - 1.th.ooni.org causes timeout
	// - 2.th.ooni.org is WAI
	// - 3.th.ooni.org causes timeout
	//
	// We expect to see a success and to never attempt with 3.th.ooni.org.
	//

	blockforever := make(chan any)

	zeroTh := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
	defer zeroTh.Close()

	oneTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockforever
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer oneTh.Close()

	expectedResponse := &apiResponse{
		Age:  41,
		Name: "sbs",
	}

	twoTh := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(must.MarshalJSON(expectedResponse))
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
	// We're splitting the algorithm into its Map step and its Reduce step because
	// this allows us to clearly observe what happened.
	//
	// Note: we're running this test with the default watchdog timeout.

	results := overlapped.Map(
		context.Background(),
		NewBaseURL(zeroTh.URL),
		NewBaseURL(oneTh.URL),
		NewBaseURL(twoTh.URL),
		NewBaseURL(threeTh.URL),
	)

	runtimex.Assert(len(results) == 4, "unexpected number of results")

	// attempt 0: should have seen connection reset
	// attempt 1: should have seen the context canceled
	// attempt 2: should be successful
	// attempt 3: should have seen the context canceled
	for _, entry := range results {
		t.Log(entry.Index, string(must.MarshalJSON(entry)))
		switch entry.Index {
		case 0:
			if err := entry.Err; !errors.Is(err, netxlite.ECONNRESET) {
				t.Fatal("unexpected error", err)
			}
		case 1, 3:
			if err := entry.Err; !errors.Is(err, context.Canceled) {
				t.Fatal("unexpected error", err)
			}
		case 2:
			if err := entry.Err; err != nil {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(expectedResponse, entry.Value); diff != "" {
				t.Fatal(diff)
			}
		default:
			t.Fatal("unexpected index", entry.Index)
		}
	}

	// Now run the reduce step of the algorithm and make sure we correctly
	// return the first success and the nil error

	apiResp, idx, err := OverlappedReduce(results)

	// we do not expect to see a failure because one of the THs is WAI
	if err != nil {
		t.Fatal(err)
	}

	if idx != 2 {
		t.Fatal("unexpected success index", idx)
	}

	// compare response to expectation
	if diff := cmp.Diff(expectedResponse, apiResp); diff != "" {
		t.Fatal(diff)
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

	apiResp, idx, err := overlapped.Run(context.Background() /* no URLs here! */)

	// we do expect to see the generic overlapped failure
	if !errors.Is(err, ErrGenericOverlappedFailure) {
		t.Fatal("unexpected error", err)
	}

	if idx != 0 {
		t.Fatal("unexpected index", idx)
	}

	// we expect a nil response
	if apiResp != nil {
		t.Fatal("expected nil API response")
	}
}

func TestNewOverlappedWithFuncDefaultsAreCorrect(t *testing.T) {
	overlapped := newOverlappedWithFunc(func(ctx context.Context, e *BaseURL) (int, error) {
		return 1, nil
	})
	if overlapped.ScheduleInterval != 15*time.Second {
		t.Fatal("unexpected ScheduleInterval")
	}
	if overlapped.WatchdogTimeout != 5*time.Minute {
		t.Fatal("unexpected WatchdogTimeout")
	}
}
