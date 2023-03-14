package aggregationapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestWorkingAsIntended(t *testing.T) {
	// expectedResp is the structure we expect to see
	expectedResp := &Response{
		Result: ResponseResult{
			AnomalyCount:     10,
			ConfirmedCount:   4,
			FailureCount:     7,
			MeasurementCount: 30,
			OKCount:          9,
		},
		V: 0,
	}

	// expectedURL is the URL we expect to see
	expectedURL := "https://www.example.com/"

	// create a test server responding to the API
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// make sure the URL is the one we expect to see
		gotURL := r.URL.Query().Get("input")
		runtimex.Assert(gotURL == expectedURL, "unexpected URL")

		// make sure the test name is the one we expect to see
		gotTestName := r.URL.Query().Get("test_name")
		runtimex.Assert(gotTestName == "web_connectivity", "unexpected test name")

		// make sure the since argument is within a reasonable range
		gotSinceStr := r.URL.Query().Get("since")
		gotSince := runtimex.Try1(time.Parse(timeFormat, gotSinceStr))
		twoMonthsAgo := time.Now().Add(-2 * 30 * 24 * time.Hour)
		runtimex.Assert(twoMonthsAgo.Before(gotSince), "too large time interval")

		data := runtimex.Try1(json.Marshal(expectedResp))
		w.Write(data)
	}))
	defer srvr.Close()

	// issue the request
	resp := Query(
		context.Background(),
		srvr.URL,
		expectedURL,
	)

	// make sure the response is correct
	if diff := cmp.Diff(expectedResp, resp); diff != "" {
		t.Fatal(diff)
	}
}
