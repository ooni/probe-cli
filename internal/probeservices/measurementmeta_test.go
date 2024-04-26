package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestGetMeasurementMeta(t *testing.T) {

	// This is the configuration we use for both testing with the real API server
	// and for testing with a local HTTP server for -short tests.
	config := model.OOAPIMeasurementMetaConfig{
		ReportID: `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`,
		Full:     true,
		Input:    `https://www.example.org`,
	}

	// This is what we expectMmeta the API to send us. We share this struct
	// because we're using it for testing with the real backend as well as
	// for testing with a local test server.
	//
	// The measurement is marked as "failed". This feels wrong but it may
	// be that the fastpah marks all urlgetter measurements as failed.
	//
	// We are not including the raw measurement for simplicity (also, there are
	// not tests for the ooni/backend API anyway, and so it's fine).
	expectMmeta := &model.OOAPIMeasurementMeta{
		Anomaly:              false,
		CategoryCode:         "",
		Confirmed:            false,
		Failure:              true,
		Input:                &config.Input,
		MeasurementStartTime: time.Date(2020, 12, 9, 5, 22, 25, 0, time.UTC),
		ProbeASN:             30722,
		ProbeCC:              "IT",
		ReportID:             "20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU",
		Scores:               `{"blocking_general":0.0,"blocking_global":0.0,"blocking_country":0.0,"blocking_isp":0.0,"blocking_local":0.0,"accuracy":0.0}`,
		TestName:             "urlgetter",
		TestStartTime:        time.Date(2020, 12, 9, 5, 22, 25, 0, time.UTC),
		RawMeasurement:       "",
	}

	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// construct a client and override the URL to be the production backend
		// instead of the testing backend, so we have stable measurements
		client := newclient()
		client.BaseURL = "https://api.ooni.io/"

		// issue the API call proper
		mmeta, err := client.GetMeasurementMeta(context.Background(), config)

		// we do not expect to see errors obviously
		if err != nil {
			t.Fatal(err)
		}

		// the raw measurement must not be empty and must parse as JSON
		//
		// once we know that, clear it for the subsequent cmp.Diff
		if mmeta.RawMeasurement == "" {
			t.Fatal("mmeta.RawMeasurement should not be empty")
		}
		var rawmeas any
		must.UnmarshalJSON([]byte(mmeta.RawMeasurement), &rawmeas)
		mmeta.RawMeasurement = ""

		// compare with the expectation
		if diff := cmp.Diff(expectMmeta, mmeta); diff != "" {
			t.Fatal(diff)
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodGet, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/measurement_meta", "invalid URL path")
			w.Write(must.MarshalJSON(expectMmeta))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the API call proper
		mmeta, err := client.GetMeasurementMeta(context.Background(), config)

		// we do not expect to see errors obviously
		if err != nil {
			t.Fatal(err)
		}

		// compare with the expectation
		if diff := cmp.Diff(expectMmeta, mmeta); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("reports an error when the connection is reset", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the API call proper
		mmeta, err := client.GetMeasurementMeta(context.Background(), config)

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect mmeta to be nil
		if mmeta != nil {
			t.Fatal("expected nil meta")
		}
	})

	t.Run("reports an error when the response is not JSON parsable", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{`))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// issue the API call proper
		mmeta, err := client.GetMeasurementMeta(context.Background(), config)

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect mmeta to be nil
		if mmeta != nil {
			t.Fatal("expected nil meta")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// issue the API call proper
		mmeta, err := client.GetMeasurementMeta(context.Background(), config)

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect mmeta to be nil
		if mmeta != nil {
			t.Fatal("expected nil meta")
		}
	})
}
