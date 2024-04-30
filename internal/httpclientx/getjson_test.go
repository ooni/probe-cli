package httpclientx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

type apiResponse struct {
	Age  int
	Name string
}

func TestGetJSON(t *testing.T) {
	t.Run("when GetRaw fails", func(t *testing.T) {
		// create a server that RST connections
		server := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// make sure the response is nil.
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})

	t.Run("when JSON parsing fails", func(t *testing.T) {
		// create a server that returns an invalid JSON type
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("[]"))
		}))
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if err.Error() != "json: cannot unmarshal array into Go value of type httpclientx.apiResponse" {
			t.Fatal("unexpected error", err)
		}

		// make sure the response is nil.
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})

	t.Run("on success", func(t *testing.T) {
		// create a server that returns a legit response
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"Name": "simone", "Age": 41}`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure the response is OK
		expect := &apiResponse{Name: "simone", Age: 41}
		if diff := cmp.Diff(expect, resp); diff != "" {
			t.Fatal(diff)
		}
	})
}

// This test ensures that GetJSON sets correct HTTP headers
func TestGetJSONHeadersOkay(t *testing.T) {
	var (
		gothost    string
		gotheaders http.Header
		gotmu      sync.Mutex
	)

	server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// save the headers
		gotmu.Lock()
		gothost = r.Host
		gotheaders = r.Header
		gotmu.Unlock()

		// send a minimal 200 Ok response
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// send the request and receive the response
	apiresp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
		Authorization: "scribai",
		Client:        http.DefaultClient,
		Host:          "www.cloudfront.com",
		Logger:        model.DiscardLogger,
		UserAgent:     model.HTTPHeaderUserAgent,
	})

	// we do not expect to see an error here
	if err != nil {
		t.Fatal(err)
	}

	// given the handler, we expect to see an empty structure here
	if apiresp.Age != 0 || apiresp.Name != "" {
		t.Fatal("expected empty response")
	}

	// make sure there are no data races
	defer gotmu.Unlock()
	gotmu.Lock()

	// make sure we have sent the authorization header
	if value := gotheaders.Get("Authorization"); value != "scribai" {
		t.Fatal("unexpected Authorization value", value)
	}

	// now make sure we have sent user-agent
	if value := gotheaders.Get("User-Agent"); value != model.HTTPHeaderUserAgent {
		t.Fatal("unexpected User-Agent value", value)
	}

	// now make sure we have sent accept-encoding
	if value := gotheaders.Get("Accept-Encoding"); value != "gzip" {
		t.Fatal("unexpected Accept-Encoding value", value)
	}

	// now make sure we could use cloudfronting
	if gothost != "www.cloudfront.com" {
		t.Fatal("unexpected Host value", gothost)
	}
}

// This test ensures GetJSON logs the response body at Debug level.
func TestGetJSONLoggingOkay(t *testing.T) {
	// create a server that returns a legit response
	server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Name": "simone", "Age": 41}`))
	}))
	defer server.Close()

	// instantiate a logger that collects logs
	logger := &testingx.Logger{}

	// invoke the API
	resp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
		Client:    http.DefaultClient,
		Logger:    logger,
		UserAgent: model.HTTPHeaderUserAgent,
	})

	t.Log(resp)
	t.Log(err)

	// make sure that the error is the expected one
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	// make sure the response is OK
	expect := &apiResponse{Name: "simone", Age: 41}
	if diff := cmp.Diff(expect, resp); diff != "" {
		t.Fatal(diff)
	}

	// collect and verify the debug lines
	debuglines := logger.DebugLines()
	t.Log(debuglines)
	if len(debuglines) != 1 {
		t.Fatal("expected to see a single debug line")
	}
	if !strings.Contains(debuglines[0], "raw response body:") {
		t.Fatal("did not see raw response body log line")
	}
}

// TestGetJSONCorrectlyRejectsNilValues ensures we correctly reject nil values.
func TestGetJSONCorrectlyRejectsNilValues(t *testing.T) {

	t.Run("when unmarshaling into a map", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`null`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[map[string]string](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if !errors.Is(err, ErrIsNil) {
			t.Fatal("unexpected error", err)
		}

		// make sure resp is nil
		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})

	t.Run("when unmarshaling into a struct pointer", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`null`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[*apiResponse](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if !errors.Is(err, ErrIsNil) {
			t.Fatal("unexpected error", err)
		}

		// make sure resp is nil
		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})

	t.Run("when unmarshaling into a slice", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`null`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := GetJSON[[]string](context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(resp)
		t.Log(err)

		// make sure that the error is the expected one
		if !errors.Is(err, ErrIsNil) {
			t.Fatal("unexpected error", err)
		}

		// make sure resp is nil
		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})
}
