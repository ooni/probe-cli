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
				&Descriptor[RawRequest, []byte]{
					Method:   http.MethodGet,
					Response: &RawResponseDescriptor{},
					URLPath:  "/",
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
					Logger: model.DiscardLogger,
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							return nil, io.EOF
						},
					},
					Logger: model.DiscardLogger,
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

		t.Run("first HTTP failure and we immediately stop", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor[RawRequest, []byte]{
					Method:   http.MethodGet,
					Response: &RawResponseDescriptor{},
					URLPath:  "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							resp := &http.Response{
								StatusCode: 403, // should cause us to return early
								Body:       io.NopCloser(strings.NewReader("deadbeef")),
							}
							return resp, nil
						},
					},
					Logger: model.DiscardLogger,
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							return nil, io.EOF
						},
					},
					Logger: model.DiscardLogger,
				},
			)
			data, idx, err := sc.Call(context.Background())
			var failure *ErrHTTPRequestFailed
			if !errors.As(err, &failure) || failure.StatusCode != 403 {
				t.Fatal("unexpected err", err)
			}
			if idx != 0 {
				t.Fatal("invalid idx")
			}
			if len(data) > 0 {
				t.Fatal("expected to see no response body")
			}
		})

		t.Run("first network failure, second success", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor[RawRequest, []byte]{
					Method:   http.MethodGet,
					Response: &RawResponseDescriptor{},
					URLPath:  "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							return nil, io.EOF // should cause us to cycle to the second entry
						},
					},
					Logger: model.DiscardLogger,
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
					Logger: model.DiscardLogger,
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

		t.Run("all network failure", func(t *testing.T) {
			sc := NewSequenceCaller(
				&Descriptor[RawRequest, []byte]{
					Method:   http.MethodGet,
					Response: &RawResponseDescriptor{},
					URLPath:  "/",
				},
				&Endpoint{
					BaseURL: "https://a.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							return nil, io.EOF // should cause us to cycle to the next entry
						},
					},
					Logger: model.DiscardLogger,
				},
				&Endpoint{
					BaseURL: "https://b.example.com/",
					HTTPClient: &mocks.HTTPClient{
						MockDo: func(req *http.Request) (*http.Response, error) {
							return nil, io.EOF // should cause us to cycle to the next entry
						},
					},
					Logger: model.DiscardLogger,
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
}
