package httpclientx

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestGetRaw(t *testing.T) {
	t.Run("when we cannot create a request", func(t *testing.T) {
		// create API call config

		rawrespbody, err := GetRaw(
			context.Background(),
			"\t", // <- invalid URL that we cannot parse
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			},
		)

		t.Log(rawrespbody)
		t.Log(err)

		if err.Error() != `parse "\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		if len(rawrespbody) != 0 {
			t.Fatal("expected zero-length body")
		}
	})

	t.Run("on success", func(t *testing.T) {
		expected := []byte(`Bonsoir, Elliot`)

		// create a server that returns a legit response
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(expected)
		}))
		defer server.Close()

		rawrespbody, err := GetRaw(context.Background(), server.URL, &Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

		t.Log(rawrespbody)
		t.Log(err)

		if err != nil {
			t.Fatal("unexpected error", err)
		}

		if diff := cmp.Diff(expected, rawrespbody); diff != "" {
			t.Fatal(diff)
		}
	})
}

// This test ensures that GetRaw sets correct HTTP headers
func TestGetRawHeadersOkay(t *testing.T) {
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
		w.Write([]byte(`<R></R>`))
	}))
	defer server.Close()

	// send the request and receive the response
	rawresp, err := GetRaw(context.Background(), server.URL, &Config{
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

	// make sure the raw response is exactly what we expect to receive
	if diff := cmp.Diff([]byte(`<R></R>`), rawresp); diff != "" {
		t.Fatal("unexpected raw response")
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

	// now make sure we could our cloudfronting
	if gothost != "www.cloudfront.com" {
		t.Fatal("unexpected Host value", gothost)
	}
}

// This test ensures GetRaw logs the response body at Debug level.
func TestGetRawLoggingOkay(t *testing.T) {
	expected := []byte(`Bonsoir, Elliot`)

	// create a server that returns a legit response
	server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(expected)
	}))
	defer server.Close()

	// instantiate a logger that collects logs
	logger := &testingx.Logger{}

	rawrespbody, err := GetRaw(context.Background(), server.URL, &Config{
		Client:    http.DefaultClient,
		Logger:    logger,
		UserAgent: model.HTTPHeaderUserAgent,
	})

	t.Log(rawrespbody)
	t.Log(err)

	if err != nil {
		t.Fatal("unexpected error", err)
	}

	if diff := cmp.Diff(expected, rawrespbody); diff != "" {
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
