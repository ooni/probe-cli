package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func Test_joinURLPath(t *testing.T) {
	tests := []struct {
		name         string
		urlPath      string
		resourcePath string
		want         string
	}{{
		name:         "whole path inside urlPath and empty resourcePath",
		urlPath:      "/robots.txt",
		resourcePath: "",
		want:         "/robots.txt",
	}, {
		name:         "empty urlPath and slash-prefixed resourcePath",
		urlPath:      "",
		resourcePath: "/foo",
		want:         "/foo",
	}, {
		name:         "slash urlPath and slash-prefixed resourcePath",
		urlPath:      "/",
		resourcePath: "/foo",
		want:         "/foo",
	}, {
		name:         "empty urlPath and empty resourcePath",
		urlPath:      "",
		resourcePath: "",
		want:         "/",
	}, {
		name:         "non-slash-terminated urlPath and slash-prefixed resourcePath",
		urlPath:      "/foo",
		resourcePath: "/bar",
		want:         "/foo/bar",
	}, {
		name:         "slash-terminated urlPath and slash-prefixed resourcePath",
		urlPath:      "/foo/",
		resourcePath: "/bar",
		want:         "/foo/bar",
	}, {
		name:         "slash-terminated urlPath and non-slash-prefixed resourcePath",
		urlPath:      "/foo",
		resourcePath: "bar",
		want:         "/foo/bar",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinURLPath(tt.urlPath, tt.resourcePath)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func Test_newRequest(t *testing.T) {
	type args struct {
		ctx      context.Context
		endpoint *Endpoint
		desc     *Descriptor
	}
	tests := []struct {
		name    string
		args    args
		wantFn  func(*testing.T, *http.Request)
		wantErr error
	}{{
		name: "url.Parse fails",
		args: args{
			ctx: nil,
			endpoint: &Endpoint{
				BaseURL:    "\t\t\t", // does not parse!
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn:  nil,
		wantErr: errors.New(`parse "\t\t\t": net/url: invalid control character in URL`),
	}, {
		name: "http.NewRequestWithContext fails",
		args: args{
			ctx: nil, // causes http.NewRequestWithContext to fail
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn:  nil,
		wantErr: errors.New("net/http: nil Context"),
	}, {
		name: "successful case with GET method, no body, and no extra headers",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			if req.Body != nil {
				t.Fatal("invalid body", req.Body)
			}
		},
		wantErr: nil,
	}, {
		name: "successful case with POST method and body",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        model.DiscardLogger,
				MaxBodySize:   0,
				Method:        http.MethodPost,
				RequestBody:   []byte("deadbeef"),
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodPost {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			data, err := netxlite.ReadAllContext(context.Background(), req.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff([]byte("deadbeef"), data); diff != "" {
				t.Fatal(diff)
			}
		},
		wantErr: nil,
	}, {
		name: "with GET method and custom headers",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "antani.org",
				UserAgent:  "httpclient/1.0.1",
			},
			desc: &Descriptor{
				Accept:        "application/json",
				Authorization: "deafbeef",
				ContentType:   "text/plain",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        http.MethodPut,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodPut {
				t.Fatal("invalid method")
			}
			if req.Host != "antani.org" {
				t.Fatal("invalid request host")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			if req.Header.Get("Authorization") != "deafbeef" {
				t.Fatal("invalid authorization")
			}
			if req.Header.Get("Content-Type") != "text/plain" {
				t.Fatal("invalid content-type")
			}
			if req.Header.Get("Accept") != "application/json" {
				t.Fatal("invalid accept")
			}
			if req.Header.Get("User-Agent") != "httpclient/1.0.1" {
				t.Fatal("invalid user-agent")
			}
		},
		wantErr: nil,
	}, {
		name: "we join the urlPath with the resourcePath",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://www.example.com/api/v1",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "/test-list/urls",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://www.example.com/api/v1/test-list/urls" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "we discard any query element inside the Endpoint.BaseURL",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.org/api/v1/?probe_cc=IT",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      map[string][]string{},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.org/api/v1/" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "we include query elements from Descriptor.URLQuery",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://www.example.com/api/v1/",
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "test-list/urls",
				URLQuery: map[string][]string{
					"probe_cc": {"IT"},
				},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://www.example.com/api/v1/test-list/urls?probe_cc=IT" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "with as many implicitly initialized fiels as possible",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
			},
			desc: &Descriptor{},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newRequest(tt.args.ctx, tt.args.endpoint, tt.args.desc)
			switch {
			case err == nil && tt.wantErr == nil:
				// nothing
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
			if tt.wantFn != nil {
				tt.wantFn(t, got)
				return
			}
			if got != nil {
				t.Fatal("got response with nil tt.wantFn")
			}
		})
	}
}

func TestCall(t *testing.T) {
	type args struct {
		ctx      context.Context
		desc     *Descriptor
		endpoint *Endpoint
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
	}{{
		name: "newRequest fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				Logger:        nil,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
			endpoint: &Endpoint{
				BaseURL:    "\t\t\t", // causes newRequest to fail
				HTTPClient: nil,
				Host:       "",
				UserAgent:  "",
			},
		},
		want:    nil,
		wantErr: errors.New(`parse "\t\t\t": net/url: invalid control character in URL`),
	}, {
		name: "endpoint.HTTPClient.Do fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
			},
		},
		want:    nil,
		wantErr: io.EOF,
	}, {
		name: "reading body fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									return 0, netxlite.ECONNRESET
								},
							}),
						}
						return resp, nil
					},
				},
			},
		},
		want:    nil,
		wantErr: errors.New(netxlite.FailureConnectionReset),
	}, {
		name: "status code indicates failure",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader("deadbeef")),
							StatusCode: 403,
						}
						return resp, nil
					},
				},
			},
		},
		want:    nil,
		wantErr: errors.New("httpapi: http request failed: 403"),
	}, {
		name: "success with log body flag",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				LogBody: true, // as documented by this test's name
				Logger:  model.DiscardLogger,
				Method:  http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader("deadbeef")),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
			},
		},
		want:    []byte("deadbeef"),
		wantErr: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Call(tt.args.ctx, tt.args.desc, tt.args.endpoint)
			switch {
			case err == nil && tt.wantErr == nil:
				// nothing
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestCallWithJSONResponse(t *testing.T) {
	type response struct {
		Name string
		Age  int64
	}
	expectedResponse := response{
		Name: "sbs",
		Age:  99,
	}
	type args struct {
		ctx      context.Context
		desc     *Descriptor
		endpoint *Endpoint
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{{
		name: "call fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
			},
			endpoint: &Endpoint{
				BaseURL: "\t\t\t\t", // causes failure
			},
		},
		wantErr: errors.New(`parse "\t\t\t\t": net/url: invalid control character in URL`),
	}, {
		name: "with good response and missing header",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader(`{"Name": "sbs", "Age": 99}`)),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
			},
		},
		wantErr: nil,
	}, {
		name: "with good response and good header",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Logger: model.DiscardLogger,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Header: http.Header{
								"Content-Type": {"application/json"},
							},
							Body:       io.NopCloser(strings.NewReader(`{"Name": "sbs", "Age": 99}`)),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
			},
		},
		wantErr: nil,
	}, {
		name: "response is not JSON",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				LogBody: false,
				Logger:  model.DiscardLogger,
				Method:  http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Header: http.Header{
								"Content-Type": {"application/json"},
							},
							Body:       io.NopCloser(strings.NewReader(`{`)), // invalid JSON
							StatusCode: 200,
						}
						return resp, nil
					},
				},
			},
		},
		wantErr: errors.New("unexpected end of JSON input"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response response
			err := CallWithJSONResponse(tt.args.ctx, tt.args.desc, tt.args.endpoint, &response)
			switch {
			case err == nil && tt.wantErr == nil:
				if diff := cmp.Diff(expectedResponse, response); err != nil {
					t.Fatal(diff)
				}
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
		})
	}
}

func TestCallHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // should fail HTTP request immediately
	desc := &Descriptor{
		LogBody: false,
		Logger:  model.DiscardLogger,
		Method:  http.MethodGet,
		URLPath: "/robots.txt",
	}
	endpoint := &Endpoint{
		BaseURL:    "https://www.example.com/",
		HTTPClient: http.DefaultClient,
		UserAgent:  model.HTTPHeaderUserAgent,
	}
	body, err := Call(ctx, desc, endpoint)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
	if len(body) > 0 {
		t.Fatal("expected zero-length body")
	}
}

func TestCallWithJSONResponseHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // should fail HTTP request immediately
	desc := &Descriptor{
		LogBody: false,
		Logger:  model.DiscardLogger,
		Method:  http.MethodGet,
		URLPath: "/robots.txt",
	}
	endpoint := &Endpoint{
		BaseURL:    "https://www.example.com/",
		HTTPClient: http.DefaultClient,
		UserAgent:  model.HTTPHeaderUserAgent,
	}
	var resp url.URL
	err := CallWithJSONResponse(ctx, desc, endpoint, &resp)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
}

func TestCallAndBodyLogging(t *testing.T) {

	// This test was originally written for the httpx package and we have adapted it
	// by keeping the ~same implementation with a custom callx function that converts
	// the previous semantics of httpx to the new semantics of httpapi.
	callx := func(baseURL string, logBody bool, logger model.Logger, request, response any) error {
		desc := MustNewPOSTJSONWithJSONResponseDescriptor(logger, "/", request).WithBodyLogging(logBody)
		runtimex.Assert(desc.LogBody == logBody, "desc.LogBody should be equal to logBody here")
		endpoint := &Endpoint{
			BaseURL:    baseURL,
			HTTPClient: http.DefaultClient,
		}
		return CallWithJSONResponse(context.Background(), desc, endpoint, response)
	}

	// we also needed to create a constructor for the logger
	newlogger := func(logs chan string) model.Logger {
		return &mocks.Logger{
			MockDebugf: func(format string, v ...interface{}) {
				logs <- fmt.Sprintf(format, v...)
			},
			MockWarnf: func(format string, v ...interface{}) {
				logs <- fmt.Sprintf(format, v...)
			},
		}
	}

	t.Run("logging enabled and 200 Ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, true, logger, input, &output)
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				found |= 1 << 1
				continue
			}
		}
		if found != (1<<0 | 1<<1) {
			t.Fatal("did not find logs")
		}
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("logging enabled and 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, true, logger, input, &output)
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				found |= 1 << 1
				continue
			}
		}
		if found != (1<<0 | 1<<1) {
			t.Fatal("did not find logs")
		}
		if !errors.Is(err, ErrRequestFailed) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("logging NOT enabled and 200 Ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, false, logger, input, &output) // no logging
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				found |= 1 << 1
				continue
			}
		}
		if found != 0 {
			t.Fatal("did find logs")
		}
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("logging NOT enabled and 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, false, logger, input, &output) // no logging
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				found |= 1 << 1
				continue
			}
		}
		if found != 0 {
			t.Fatal("did find logs")
		}
		if !errors.Is(err, ErrRequestFailed) {
			t.Fatal("unexpected err", err)
		}
	})
}
