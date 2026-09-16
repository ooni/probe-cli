package probeservices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// newMeasurementForSubmit returns a measurement with the minimal fields that we
// need for submitting it to the submit_measurement endpoint.
func newMeasurementForSubmit() *model.Measurement {
	return &model.Measurement{
		DataFormatVersion:    model.OOAPIReportDefaultDataFormatVersion,
		MeasurementStartTime: "2019-10-28 12:51:07",
		ProbeASN:             "AS117",
		ProbeCC:              "IT",
		SoftwareName:         "ooniprobe-engine",
		SoftwareVersion:      "0.1.0",
		TestKeys:             map[string]any{"failure": nil},
		TestName:             "dummy",
		TestStartTime:        "2019-10-28 12:51:06",
		TestVersion:          "0.1.0",
	}
}

// newSubmitRequest returns the request that we expect the client to send for the
// given measurement: the serialized measurement wrapped into the JSON envelope.
func newSubmitRequest(t *testing.T, m *model.Measurement) *model.OOAPISubmitMeasurementRequest {
	content, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return &model.OOAPISubmitMeasurementRequest{
		Format:  "json",
		Content: string(content),
	}
}

func TestSubmit(t *testing.T) {
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create client
		client := newclient()

		// submit a measurement, which we really don't want to do in short mode
		muid, err := client.Submit(context.Background(), newMeasurementForSubmit())

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect the backend to assign a measurement UID
		if muid == "" {
			t.Fatal("expected a non-empty measurement UID")
		}
	})

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// this is the measurement we're going to submit
		measurement := newMeasurementForSubmit()

		// this is the request we expect the client to send
		expectreq := newSubmitRequest(t, measurement)

		// this is what we expect to receive
		expect := &model.OOAPISubmitMeasurementResponse{
			MeasurementUID: "20240301000000.000000_IT_dummy_0123456789abcdef",
		}

		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/submit_measurement", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPISubmitMeasurementRequest
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(expectreq, &gotrequest)
			runtimex.Assert(diff == "", "request mismatch:"+diff)
			w.Write(must.MarshalJSON(expect))
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

		// submit the measurement
		muid, err := client.Submit(context.Background(), measurement)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect the measurement UID sent by the server
		if muid != expect.MeasurementUID {
			t.Fatal("unexpected measurement UID", muid)
		}
	})

	t.Run("we can use cloudfronting", func(t *testing.T) {
		// this is the measurement we're going to submit
		measurement := newMeasurementForSubmit()

		// this is the request we expect the client to send
		expectreq := newSubmitRequest(t, measurement)

		// this is what we expect to receive
		expect := &model.OOAPISubmitMeasurementResponse{
			MeasurementUID: "20240301000000.000000_IT_dummy_0123456789abcdef",
		}

		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Host == "www.cloudfront.com", "invalid r.Host")
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/submit_measurement", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPISubmitMeasurementRequest
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(expectreq, &gotrequest)
			runtimex.Assert(diff == "", "request mismatch:"+diff)
			w.Write(must.MarshalJSON(expect))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// make sure we're using cloudfronting
		client.Host = "www.cloudfront.com"

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

		// submit the measurement
		muid, err := client.Submit(context.Background(), measurement)

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect the measurement UID sent by the server
		if muid != expect.MeasurementUID {
			t.Fatal("unexpected measurement UID", muid)
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

		// submit the measurement
		muid, err := client.Submit(context.Background(), newMeasurementForSubmit())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect an empty measurement UID
		if muid != "" {
			t.Fatal("expected an empty measurement UID")
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

		// submit the measurement
		muid, err := client.Submit(context.Background(), newMeasurementForSubmit())

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect an empty measurement UID
		if muid != "" {
			t.Fatal("expected an empty measurement UID")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// submit the measurement
		muid, err := client.Submit(context.Background(), newMeasurementForSubmit())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect an empty measurement UID
		if muid != "" {
			t.Fatal("expected an empty measurement UID")
		}
	})

	t.Run("correctly handles the case where the measurement is not serializable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// create a measurement whose test keys cannot be serialized
		measurement := newMeasurementForSubmit()
		measurement.TestKeys = make(chan int)

		// submit the measurement
		muid, err := client.Submit(context.Background(), measurement)

		// we do expect an error
		if err == nil {
			t.Fatal("expected an error here")
		}

		// we expect an empty measurement UID
		if muid != "" {
			t.Fatal("expected an empty measurement UID")
		}
	})
}
