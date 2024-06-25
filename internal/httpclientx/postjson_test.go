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
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

type apiRequest struct {
	UserID int
}

func TestPostJSON(t *testing.T) {
	t.Run("when we cannot marshal the request body", func(t *testing.T) {
		// a channel cannot be serialized
		req := make(chan int)
		close(req)

		resp, err := PostJSON[chan int, *apiResponse](
			context.Background(),
			NewBaseURL(""),
			req,
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			})

		t.Log(resp)
		t.Log(err)

		if err.Error() != `json: unsupported type: chan int` {
			t.Fatal("unexpected error", err)
		}

		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})

	t.Run("when we cannot create a request", func(t *testing.T) {
		req := &apiRequest{117}

		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL("\t"), // <- invalid URL that we cannot parse
			req,
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			},
		)

		t.Log(resp)
		t.Log(err)

		if err.Error() != `parse "\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})

	t.Run("in case of HTTP failure", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server.Close()

		req := &apiRequest{117}

		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			req,
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			})

		t.Log(resp)
		t.Log(err)

		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		if resp != nil {
			t.Fatal("expected nil resp")
		}
	})

	t.Run("when we cannot parse the response body", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("[]"))
		}))
		defer server.Close()

		req := &apiRequest{117}

		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			req,
			&Config{
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
		req := &apiRequest{117}

		expect := &apiResponse{Name: "simone", Age: 41}

		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var gotreq apiRequest
			data := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			must.UnmarshalJSON(data, &gotreq)
			if gotreq.UserID != req.UserID {
				w.WriteHeader(404)
				return
			}
			w.Write(must.MarshalJSON(expect))
		}))
		defer server.Close()

		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			req,
			&Config{
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

		// make sure the response is OK.
		if diff := cmp.Diff(expect, resp); diff != "" {
			t.Fatal(diff)
		}
	})
}

// This test ensures that PostJSON sets correct HTTP headers and sends the right body.
func TestPostJSONCommunicationOkay(t *testing.T) {
	var (
		gothost    string
		gotheaders http.Header
		gotrawbody []byte
		gotmu      sync.Mutex
	)

	server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read the raw response body
		rawbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))

		// save the raw response body and headers
		gotmu.Lock()
		gothost = r.Host
		gotrawbody = rawbody
		gotheaders = r.Header
		gotmu.Unlock()

		// send a minimal 200 Ok response
		w.WriteHeader(200)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// create and serialize the expected request body
	apireq := &apiRequest{
		UserID: 117,
	}
	rawapireq := must.MarshalJSON(apireq)

	// send the request and receive the response
	apiresp, err := PostJSON[*apiRequest, *apiResponse](
		context.Background(),
		NewBaseURL(server.URL).WithHostOverride("www.cloudfront.com"),
		apireq,
		&Config{
			Authorization: "scribai",
			Client:        http.DefaultClient,
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

	// now verify what the handler has read as the raw request body
	if diff := cmp.Diff(rawapireq, gotrawbody); diff != "" {
		t.Fatal(diff)
	}

	// make sure we have sent the authorization header
	if value := gotheaders.Get("Authorization"); value != "scribai" {
		t.Fatal("unexpected Authorization value", value)
	}

	// now make sure we have sent content-type
	if value := gotheaders.Get("Content-Type"); value != "application/json" {
		t.Fatal("unexpected Content-Type value", value)
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

// This test ensures PostJSON logs the request and response body at Debug level.
func TestPostJSONLoggingOkay(t *testing.T) {
	req := &apiRequest{117}

	expect := &apiResponse{Name: "simone", Age: 41}

	server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gotreq apiRequest
		data := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
		must.UnmarshalJSON(data, &gotreq)
		if gotreq.UserID != req.UserID {
			w.WriteHeader(404)
			return
		}
		w.Write(must.MarshalJSON(expect))
	}))
	defer server.Close()

	// instantiate a logger that collects logs
	logger := &testingx.Logger{}

	resp, err := PostJSON[*apiRequest, *apiResponse](
		context.Background(),
		NewBaseURL(server.URL),
		req,
		&Config{
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

	// make sure the response is OK.
	if diff := cmp.Diff(expect, resp); diff != "" {
		t.Fatal(diff)
	}

	// collect and verify the debug lines
	debuglines := logger.DebugLines()
	t.Log(debuglines)
	if len(debuglines) != 2 {
		t.Fatal("expected to see a single debug line")
	}
	if !strings.Contains(debuglines[0], "raw request body:") {
		t.Fatal("did not see raw request body log line")
	}
	if !strings.Contains(debuglines[1], "raw response body:") {
		t.Fatal("did not see raw response body log line")
	}
}

// TestPostJSONCorrectlyRejectsNilValues ensures we do not emit and correctly reject nil values.
func TestPostJSONCorrectlyRejectsNilValues(t *testing.T) {

	t.Run("when sending a nil map", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := PostJSON[map[string]string, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			nil,
			&Config{
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

	t.Run("when sending a nil struct pointer", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			nil,
			&Config{
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

	t.Run("when sending a nil slice", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		// invoke the API
		resp, err := PostJSON[[]string, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			nil,
			&Config{
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

	t.Run("when unmarshaling into a map", func(t *testing.T) {
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`null`))
		}))
		defer server.Close()

		// create an empty request
		apireq := &apiRequest{}

		// invoke the API
		resp, err := PostJSON[*apiRequest, map[string]string](
			context.Background(),
			NewBaseURL(server.URL),
			apireq,
			&Config{
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

		// create an empty request
		apireq := &apiRequest{}

		// invoke the API
		resp, err := PostJSON[*apiRequest, *apiResponse](
			context.Background(),
			NewBaseURL(server.URL),
			apireq,
			&Config{
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

		// create an empty request
		apireq := &apiRequest{}

		// invoke the API
		resp, err := PostJSON[*apiRequest, []string](
			context.Background(),
			NewBaseURL(server.URL),
			apireq,
			&Config{
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
