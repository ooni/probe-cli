package httpapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSequenceCaller(t *testing.T) {
	t.Run("Call", func(t *testing.T) {
		t.Run("first success", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader("deadbeef")),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader("abad1dea")), // different
							}
							return resp, nil
						},
					},
				},
			)
			data, idx, err := sc.Call(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if idx != 0 {
				t.Fatal("invalid idx")
			}
			if diff := cmp.Diff([]byte("deadbeef"), data); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("first failure, second success", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader("deadbeef")),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader("abad1dea")),
							}
							return resp, nil
						},
					},
				},
			)
			data, idx, err := sc.Call(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if idx != 1 {
				t.Fatal("invalid idx")
			}
			if diff := cmp.Diff([]byte("abad1dea"), data); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("all failure", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader("deadbeef")),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader("abad1dea")),
							}
							return resp, nil
						},
					},
				},
			)
			data, idx, err := sc.Call(context.Background())
			if !errors.Is(err, ErrAllEndpointsFailed) {
				t.Fatal("unexpected err", err)
			}
			if idx != -1 {
				t.Fatal("invalid idx")
			}
			if len(data) > 0 {
				t.Fatal("expected zero-length data")
			}
		})
	})

	t.Run("CallWithJSONResponse", func(t *testing.T) {
		type response struct {
			Name string
			Age  int64
		}

		t.Run("first success", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader(`{"Name":"sbs","Age":99}`)),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader(`{}`)), // different
							}
							return resp, nil
						},
					},
				},
			)
			expect := response{
				Name: "sbs",
				Age:  99,
			}
			var got response
			idx, err := sc.CallWithJSONResponse(context.Background(), &got)
			if err != nil {
				t.Fatal(err)
			}
			if idx != 0 {
				t.Fatal("invalid idx")
			}
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("first failure, second success", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader(`{}`)),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 200,
								Body:       io.NopCloser(strings.NewReader(`{"Age":155}`)),
							}
							return resp, nil
						},
					},
				},
			)
			expect := response{
				Name: "",
				Age:  155,
			}
			var got response
			idx, err := sc.CallWithJSONResponse(context.Background(), &got)
			if err != nil {
				t.Fatal(err)
			}
			if idx != 1 {
				t.Fatal("invalid idx")
			}
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("all failure", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor{
					Logger:  model.DiscardLogger,
					Method:  http.MethodGet,
					URLPath: "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader(`{"Age": 144}`)),
							}
							return resp, nil
						},
					},
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should be enough to cause us to call the second entry
								Body:       io.NopCloser(strings.NewReader(`{"Age": 177}`)),
							}
							return resp, nil
						},
					},
				},
			)
			var got response
			idx, err := sc.CallWithJSONResponse(context.Background(), &got)
			if !errors.Is(err, ErrAllEndpointsFailed) {
				t.Fatal("unexpected err", err)
			}
			if idx != -1 {
				t.Fatal("invalid idx")
			}
		})
	})
}
