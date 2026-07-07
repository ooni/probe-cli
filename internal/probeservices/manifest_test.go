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

// sampleManifest
const sampleManifest = `{
  "manifest": {
    "nym_scope": "ooni.org/{probe_cc}/{probe_asn}",
    "submission_policy": [
      {
        "policy": {
          "age": [2461110, 2826140],
          "min_measurement_count": 0
        },
        "match": {
          "probe_cc": "*",
          "probe_asn": "*"
        }
      }
    ],
    "public_parameters": "AdqzxWc0xFMFlXygX+KfKxRGy6EE"
  },
  "meta": {
    "version": "TjxIhQyJHRZsqmidU_coSEl2dZUiBGvL",
    "protocol_version": "0.1.0"
  }
}`

// newSampleManifest returns the same manifest as [sampleManifest] as a struct,
// so tests can serve it from a local server and diff the round trip.
func newSampleManifest() *model.OOAPIManifest {
	return &model.OOAPIManifest{
		Manifest: model.OOAPIManifestBody{
			NymScope: "ooni.org/{probe_cc}/{probe_asn}",
			SubmissionPolicy: []model.OOAPISubmissionPolicy{{
				Policy: model.OOAPISubmissionPolicyParams{
					Age:                 []uint32{2461110, 2826140},
					MinMeasurementCount: 0,
				},
				Match: model.OOAPISubmissionPolicyMatch{
					ProbeCC:  "*",
					ProbeASN: "*",
				},
			}},
			PublicParameters: "AdqzxWc0xFMFlXygX+KfKxRGy6EE",
		},
		Meta: model.OOAPIManifestMeta{
			Version:         "TjxIhQyJHRZsqmidU_coSEl2dZUiBGvL",
			ProtocolVersion: "0.1.0",
		},
	}
}

func parseSampleManifest(t *testing.T) *model.OOAPIManifest {
	var manifest model.OOAPIManifest
	if err := json.Unmarshal([]byte(sampleManifest), &manifest); err != nil {
		t.Fatal(err)
	}
	return &manifest
}

func TestManifestParsing(t *testing.T) {
	manifest := parseSampleManifest(t)
	if manifest.Meta.Version != "TjxIhQyJHRZsqmidU_coSEl2dZUiBGvL" {
		t.Fatal("unexpected version", manifest.Meta.Version)
	}
	if manifest.Manifest.PublicParameters == "" {
		t.Fatal("missing public parameters")
	}
	if len(manifest.Manifest.SubmissionPolicy) != 1 {
		t.Fatal("unexpected policy count")
	}
	if got := manifest.Manifest.SubmissionPolicy[0].Policy.Age; len(got) != 2 {
		t.Fatal("unexpected age length", got)
	}
}

func TestGetManifest(t *testing.T) {
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create client
		client := newclient()

		// issue the request
		manifest, err := client.GetManifest(context.Background())

		// we do not expect an error here
		if err != nil {
			t.Fatal(err)
		}

		// we expect a usable manifest
		if manifest.Meta.Version == "" {
			t.Fatal("expected a non-empty manifest version")
		}
		if manifest.Manifest.PublicParameters == "" {
			t.Fatal("expected non-empty public parameters")
		}
		if len(manifest.Manifest.SubmissionPolicy) < 1 {
			t.Fatal("expected at least one submission policy")
		}
	})

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// this is what we expect to receive
		expect := newSampleManifest()

		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Method == http.MethodGet, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/manifest", "invalid URL path")
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

		// issue the GET request
		manifest, err := client.GetManifest(context.Background())

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, manifest); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we can use cloudfronting", func(t *testing.T) {
		// this is what we expect to receive
		expect := newSampleManifest()

		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			runtimex.Assert(r.Host == "www.cloudfront.com", "invalid r.Host")
			runtimex.Assert(r.Method == http.MethodGet, "invalid method")
			runtimex.Assert(r.URL.Path == "/api/v1/manifest", "invalid URL path")
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

		// issue the GET request
		manifest, err := client.GetManifest(context.Background())

		// we do not expect an error
		if err != nil {
			t.Fatal(err)
		}

		// we expect to see exactly what the server sent
		if diff := cmp.Diff(expect, manifest); diff != "" {
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

		// issue the GET request
		manifest, err := client.GetManifest(context.Background())

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil manifest
		if manifest != nil {
			t.Fatal("expected nil manifest")
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

		// issue the GET request
		manifest, err := client.GetManifest(context.Background())

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil manifest
		if manifest != nil {
			t.Fatal("expected nil manifest")
		}
	})

	t.Run("correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// issue the GET request
		manifest, err := client.GetManifest(context.Background())

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil manifest
		if manifest != nil {
			t.Fatal("expected nil manifest")
		}
	})
}

func TestGetRangesFromPolicy(t *testing.T) {

	t.Run("matches a wildcard policy for any probe", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		ageRange, minMeasurementCount, err := GetRangesFromPolicy(manifest, "IT", "AS117")
		if err != nil {
			t.Fatal(err)
		}
		if ageRange[0] != 2461110 || ageRange[1] != 2826140 {
			t.Fatal("unexpected age range", ageRange)
		}
		if minMeasurementCount != 0 {
			t.Fatal("unexpected min measurement count", minMeasurementCount)
		}
	})

	t.Run("matches an exact probe and rejects a different one", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		manifest.Manifest.SubmissionPolicy[0].Match = model.OOAPISubmissionPolicyMatch{
			ProbeCC:  "IT",
			ProbeASN: "AS117",
		}

		if _, _, err := GetRangesFromPolicy(manifest, "IT", "AS117"); err != nil {
			t.Fatal("expected a match for the exact probe", err)
		}

		if _, _, err := GetRangesFromPolicy(manifest, "US", "AS117"); !errors.Is(err, ErrNoMatchingPolicy) {
			t.Fatalf("not the error we expected: %+v", err)
		}

		if _, _, err := GetRangesFromPolicy(manifest, "IT", "AS30722"); !errors.Is(err, ErrNoMatchingPolicy) {
			t.Fatalf("not the error we expected: %+v", err)
		}
	})

	t.Run("matches when only one field is wildcarded", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		manifest.Manifest.SubmissionPolicy[0].Match = model.OOAPISubmissionPolicyMatch{
			ProbeCC:  "IT",
			ProbeASN: "*",
		}
		if _, _, err := GetRangesFromPolicy(manifest, "IT", "AS999"); err != nil {
			t.Fatal("expected a match for CC-exact + ASN-wildcard", err)
		}
		if _, _, err := GetRangesFromPolicy(manifest, "US", "AS999"); !errors.Is(err, ErrNoMatchingPolicy) {
			t.Fatalf("not the error we expected: %+v", err)
		}
	})

	t.Run("returns the first matching policy (highest priority first)", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		// Prepend a more specific policy so it should win over the wildcard one.
		specific := model.OOAPISubmissionPolicy{
			Policy: model.OOAPISubmissionPolicyParams{
				Age:                 []uint32{100, 200},
				MinMeasurementCount: 5,
			},
			Match: model.OOAPISubmissionPolicyMatch{ProbeCC: "IT", ProbeASN: "AS117"},
		}
		manifest.Manifest.SubmissionPolicy = append(
			[]model.OOAPISubmissionPolicy{specific}, manifest.Manifest.SubmissionPolicy...)

		ageRange, minMeasurementCount, err := GetRangesFromPolicy(manifest, "IT", "AS117")
		if err != nil {
			t.Fatal(err)
		}
		if ageRange[0] != 100 || ageRange[1] != 200 || minMeasurementCount != 5 {
			t.Fatal("expected the first (specific) policy to win", ageRange, minMeasurementCount)
		}

		// A probe not matching the specific policy falls through to the wildcard.
		ageRange, minMeasurementCount, err = GetRangesFromPolicy(manifest, "US", "AS999")
		if err != nil {
			t.Fatal(err)
		}
		if ageRange[0] != 2461110 || ageRange[1] != 2826140 || minMeasurementCount != 0 {
			t.Fatal("expected the wildcard policy for a non-specific probe", ageRange, minMeasurementCount)
		}
	})

	t.Run("returns no matching policy error when the list is empty", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		manifest.Manifest.SubmissionPolicy = nil
		if _, _, err := GetRangesFromPolicy(manifest, "IT", "AS117"); !errors.Is(err, ErrNoMatchingPolicy) {
			t.Fatalf("not the error we expected: %+v", err)
		}
	})

	t.Run("errors when the matched policy has an invalid age range", func(t *testing.T) {
		manifest := parseSampleManifest(t)
		manifest.Manifest.SubmissionPolicy[0].Policy.Age = []uint32{123} // too short
		_, _, err := GetRangesFromPolicy(manifest, "IT", "AS117")
		if err == nil || errors.Is(err, ErrNoMatchingPolicy) {
			t.Fatal("expected an invalid-age-range error, got", err)
		}
	})
}
