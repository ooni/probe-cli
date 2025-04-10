package probeservices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestCheckIn(t *testing.T) {
	// define a common configuration to use across all tests
	config := model.OOAPICheckInConfig{
		Charging:        true,
		OnWiFi:          true,
		Platform:        "android",
		ProbeASN:        "AS12353",
		ProbeCC:         "PT",
		RunType:         model.RunTypeTimed,
		SoftwareName:    "ooniprobe-android",
		SoftwareVersion: "2.7.1",
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{"NEWS", "CULTR"},
		},
	}

	t.Run("with the real API server", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		client := newclient()
		client.BaseURL = "https://api.dev.ooni.io" // use the test infra

		ctx := context.Background()

		// call the API
		result, err := client.CheckIn(ctx, config)

		// we do not expect to see an error
		if err != nil {
			t.Fatal(err)
		}

		// sanity check the returned response
		if result == nil || result.Tests.WebConnectivity == nil {
			t.Fatal("got nil result or nil WebConnectivity")
		}
		if result.Tests.WebConnectivity.ReportID == "" {
			t.Fatal("ReportID is empty")
		}
		if len(result.Tests.WebConnectivity.URLs) < 1 {
			t.Fatal("unexpected number of URLs")
		}

		// ensure the category codes match our request
		for _, entry := range result.Tests.WebConnectivity.URLs {
			if entry.CategoryCode != "NEWS" && entry.CategoryCode != "CULTR" {
				t.Fatalf("unexpected category code: %+v", entry)
			}
		}
	})

	t.Run("with a working-as-intended local server", func(t *testing.T) {
		// define our expectations
		expect := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{},
				TestHelpers: map[string][]model.OOAPIService{
					"web-connectivity": {{
						Address: "https://0.th.ooni.org/",
						Type:    "https",
					}},
				},
			},
			ProbeASN: "AS30722",
			ProbeCC:  "US",
			Tests: model.OOAPICheckInResultNettests{
				WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
					ReportID: "20240424T134700Z_webconnectivity_IT_30722_n1_q5N5YSTWEqHYDo9v",
					URLs: []model.OOAPIURLInfo{{
						CategoryCode: "NEWS",
						CountryCode:  "IT",
						URL:          "https://www.example.com/",
					}},
				},
			},
			UTCTime: time.Date(2022, 11, 22, 1, 2, 3, 0, time.UTC),
			V:       1,
		}

		// create a local server that responds with the expectation
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/check-in", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPICheckInConfig
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(config, gotrequest)
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

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do not expect to see an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, result); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we can use cloudfronting", func(t *testing.T) {
		// define our expectations
		expect := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{},
				TestHelpers: map[string][]model.OOAPIService{
					"web-connectivity": {{
						Address: "https://0.th.ooni.org/",
						Type:    "https",
					}},
				},
			},
			ProbeASN: "AS30722",
			ProbeCC:  "US",
			Tests: model.OOAPICheckInResultNettests{
				WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
					ReportID: "20240424T134700Z_webconnectivity_IT_30722_n1_q5N5YSTWEqHYDo9v",
					URLs: []model.OOAPIURLInfo{{
						CategoryCode: "NEWS",
						CountryCode:  "IT",
						URL:          "https://www.example.com/",
					}},
				},
			},
			UTCTime: time.Date(2022, 11, 22, 1, 2, 3, 0, time.UTC),
			V:       1,
		}

		// create a local server that responds with the expectation
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Host == "www.cloudfront.com", "invalid r.Host")
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/check-in", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPICheckInConfig
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(config, gotrequest)
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

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do not expect to see an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, result); diff != "" {
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

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a nil pointer
		if result != nil {
			t.Fatal("expected result to be nil")
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

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a nil pointer
		if result != nil {
			t.Fatal("expected result to be nil")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect to see a nil pointer
		if result != nil {
			t.Fatal("expected result to be nil")
		}
	})

	t.Run("we store feature flags coming from the check-in API", func(t *testing.T) {
		// define our expectations
		//
		// note: when calling the checkincache.GetFeatureFlag function, we will
		// get a false result if there's no corresponding information into the
		// key-value store. So, we're setting true here to check we can read from
		// the key-value store values that are different from the default.
		expect := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{
					"torsf_enabled":       true,
					"vanilla_tor_enabled": true,
				},
				TestHelpers: map[string][]model.OOAPIService{
					"web-connectivity": {{
						Address: "https://0.th.ooni.org/",
						Type:    "https",
					}},
				},
			},
			ProbeASN: "AS30722",
			ProbeCC:  "US",
			Tests: model.OOAPICheckInResultNettests{
				WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
					ReportID: "20240424T134700Z_webconnectivity_IT_30722_n1_q5N5YSTWEqHYDo9v",
					URLs: []model.OOAPIURLInfo{{
						CategoryCode: "NEWS",
						CountryCode:  "IT",
						URL:          "https://www.example.com/",
					}},
				},
			},
			UTCTime: time.Date(2022, 11, 22, 1, 2, 3, 0, time.UTC),
			V:       1,
		}

		// create a local server that responds with the expectation
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/check-in", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPICheckInConfig
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(config, gotrequest)
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

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do not expect to see an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, result); diff != "" {
			t.Fatal(diff)
		}

		// make sure we have written non-default values into the key-value store
		if !checkincache.GetFeatureFlag(client.KVStore, "torsf_enabled", false) {
			t.Fatal("expected to see true here")
		}
		if !checkincache.GetFeatureFlag(client.KVStore, "vanilla_tor_enabled", false) {
			t.Fatal("expected to see true here")
		}
		checkinrawdata, err := client.KVStore.Get(checkincache.CheckInFlagsState)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(checkinrawdata))
		var checkindata map[string]any
		if err := json.Unmarshal(checkinrawdata, &checkindata); err != nil {
			t.Fatal(err)
		}
		t.Log(checkindata)
	})

	t.Run("does not fail if the key-value store fails", func(t *testing.T) {
		// define our expectations
		//
		// note: when calling the checkincache.GetFeatureFlag function, we will
		// get a false result if there's no corresponding information into the
		// key-value store. So, we're setting true here to check we can read from
		// the key-value store values that are different from the default.
		expect := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{
					"torsf_enabled":       true,
					"vanilla_tor_enabled": true,
				},
				TestHelpers: map[string][]model.OOAPIService{
					"web-connectivity": {{
						Address: "https://0.th.ooni.org/",
						Type:    "https",
					}},
				},
			},
			ProbeASN: "AS30722",
			ProbeCC:  "US",
			Tests: model.OOAPICheckInResultNettests{
				WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
					ReportID: "20240424T134700Z_webconnectivity_IT_30722_n1_q5N5YSTWEqHYDo9v",
					URLs: []model.OOAPIURLInfo{{
						CategoryCode: "NEWS",
						CountryCode:  "IT",
						URL:          "https://www.example.com/",
					}},
				},
			},
			UTCTime: time.Date(2022, 11, 22, 1, 2, 3, 0, time.UTC),
			V:       1,
		}

		// create a local server that responds with the expectation
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodPost, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/check-in", "invalid URL path")
			rawreqbody := runtimex.Try1(netxlite.ReadAllContext(r.Context(), r.Body))
			var gotrequest model.OOAPICheckInConfig
			must.UnmarshalJSON(rawreqbody, &gotrequest)
			diff := cmp.Diff(config, gotrequest)
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

		// override the key-value store
		//
		// note: we create a key-value store that cannot store and we want to check
		// that (1) still the check-in call succeeds and (2) however, we have not
		// store anything interesting inside the check-in call.
		goodKvStore := client.KVStore
		brokenKvStore := &mocks.KeyValueStore{
			MockGet: goodKvStore.Get,
			MockSet: func(key string, value []byte) error {
				return errors.New("mocked error")
			},
		}
		client.KVStore = brokenKvStore

		// call the API
		result, err := client.CheckIn(context.Background(), config)

		// we do not expect to see an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, result); diff != "" {
			t.Fatal(diff)
		}

		// make sure we are still getting the default values here
		if checkincache.GetFeatureFlag(client.KVStore, "torsf_enabled", false) {
			t.Fatal("expected to see false here")
		}
		if checkincache.GetFeatureFlag(client.KVStore, "vanilla_tor_enabled", false) {
			t.Fatal("expected to see false here")
		}
		checkinrawdata, err := client.KVStore.Get(checkincache.CheckInFlagsState)
		if !errors.Is(err, kvstore.ErrNoSuchKey) {
			t.Fatal("unexpected error", err)
		}
		if len(checkinrawdata) != 0 {
			t.Fatal("expected zero-length data here")
		}
	})
}
