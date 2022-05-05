package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/fakefill"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// userAgent is the user agent used by this test suite
var userAgent = fmt.Sprintf("ooniprobe-cli/%s", version.Version)

func TestAPIClientTemplate(t *testing.T) {
	t.Run("WithBodyLogging", func(t *testing.T) {
		tmpl := &APIClientTemplate{
			HTTPClient: http.DefaultClient,
			LogBody:    false, // explicit default initialization for clarity
			Logger:     model.DiscardLogger,
		}
		child := tmpl.WithBodyLogging()
		if !child.LogBody {
			t.Fatal("expected body logging to be enabled")
		}
		if tmpl.LogBody {
			t.Fatal("expected body logging to still be disabled")
		}
	})

	t.Run("normal constructor", func(t *testing.T) {
		// Implementation note: the fakefiller will ignore the
		// fields it does not know how to fill, so we are filling
		// those fields with plausible values in advance
		tmpl := &APIClientTemplate{
			HTTPClient: http.DefaultClient,
			Logger:     model.DiscardLogger,
		}
		ff := &fakefill.Filler{}
		ff.Fill(tmpl)
		ac := tmpl.Build()
		orig := apiClient(*tmpl)
		if diff := cmp.Diff(&orig, ac); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("constructor with authorization", func(t *testing.T) {
		// Implementation note: the fakefiller will ignore the
		// fields it does not know how to fill, so we are filling
		// those fields with plausible values in advance
		tmpl := &APIClientTemplate{
			HTTPClient: http.DefaultClient,
			Logger:     model.DiscardLogger,
		}
		ff := &fakefill.Filler{}
		ff.Fill(tmpl)
		tok := ""
		ff.Fill(&tok)
		ac := tmpl.BuildWithAuthorization(tok)
		// the authorization should be different now
		if tmpl.Authorization == ac.(*apiClient).Authorization {
			t.Fatal("we expect Authorization to be different")
		}
		// clear authorization for the comparison
		tmpl.Authorization = ""
		ac.(*apiClient).Authorization = ""
		orig := apiClient(*tmpl)
		if diff := cmp.Diff(&orig, ac); diff != "" {
			t.Fatal(diff)
		}
	})
}

// newAPIClient is an helper factory creating a client for testing.
func newAPIClient() *apiClient {
	return &apiClient{
		BaseURL:    "https://example.com",
		HTTPClient: http.DefaultClient,
		Logger:     model.DiscardLogger,
		UserAgent:  userAgent,
	}
}

func TestJoinURLPath(t *testing.T) {
	t.Run("with Base URL and path", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com"
		if got, want := ac.joinURLPath("/foo"), ""; got == want {
			t.Fatal("expected result")
		}
	})

	t.Run("both Base URL and URL path contains /", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com/"
		if got, want := ac.joinURLPath("/foo"), "http://example.com/foo"; got != want {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with empty URL path", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com"
		if got, want := ac.joinURLPath(""), "http://example.com"; got != want {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with no BaseURL", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = ""
		if got, want := ac.joinURLPath("/foo"), ""; got != want {
			t.Fatal("unexpected result")
		}
	})

	t.Run("URL path with the BaseURL", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com/foo"
		if got, want := ac.joinURLPath("/bar"), "http://example.com/foo/bar"; got != want {
			t.Fatal("unexpected result")
		}
	})

	t.Run("URL path with the BaseURL and slash", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com/foo/"
		if got, want := ac.joinURLPath("/bar"), "http://example.com/foo/bar"; got != want {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with the BaseURL slash and no slash in URL path", func(t *testing.T) {
		ac := newAPIClient()
		ac.BaseURL = "http://example.com/foo/"
		if got, want := ac.joinURLPath("bar"), "http://example.com/foo/bar"; got != want {
			t.Fatal("unexpected result")
		}
	})
}

// fakeRequest is a fake request we serialize.
type fakeRequest struct {
	Name       string
	Age        int
	Sleeping   bool
	Attributes map[string][]string
}

func TestAPIClient(t *testing.T) {
	t.Run("newRequestWithJSONBody", func(t *testing.T) {
		t.Run("JSON marshal failure", func(t *testing.T) {
			client := newAPIClient()
			req, err := client.newRequestWithJSONBody(
				context.Background(), "GET", "/", nil, make(chan interface{}),
			)
			if err == nil || !strings.HasPrefix(err.Error(), "json: unsupported type") {
				t.Fatal("not the error we expected", err)
			}
			if req != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("newRequest failure", func(t *testing.T) {
			client := newAPIClient()
			client.BaseURL = "\t\t\t" // cause URL parse error
			req, err := client.newRequestWithJSONBody(
				context.Background(), "GET", "/", nil, nil,
			)
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("not the error we expected")
			}
			if req != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("sets the content-type properly", func(t *testing.T) {
			var jsonReq fakeRequest
			ff := &fakefill.Filler{}
			ff.Fill(&jsonReq)
			client := newAPIClient()
			req, err := client.newRequestWithJSONBody(
				context.Background(), "GET", "/", nil, jsonReq,
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatal("did not set content-type properly")
			}
		})
	})

	t.Run("newRequest", func(t *testing.T) {
		t.Run("with invalid method", func(t *testing.T) {
			client := newAPIClient()
			req, err := client.newRequest(
				context.Background(), "\t\t\t", "/", nil, nil,
			)
			if err == nil || !strings.HasPrefix(err.Error(), "net/http: invalid method") {
				t.Fatal("not the error we expected")
			}
			if req != nil {
				t.Fatal("expected nil request here")
			}
		})

		t.Run("with query", func(t *testing.T) {
			client := newAPIClient()
			q := url.Values{}
			q.Add("antani", "mascetti")
			q.Add("melandri", "conte")
			req, err := client.newRequest(
				context.Background(), "GET", "/", q, nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.URL.Query().Get("antani") != "mascetti" {
				t.Fatal("expected different query string here")
			}
			if req.URL.Query().Get("melandri") != "conte" {
				t.Fatal("expected different query string here")
			}
		})

		t.Run("with authorization", func(t *testing.T) {
			client := newAPIClient()
			client.Authorization = "deadbeef"
			req, err := client.newRequest(
				context.Background(), "GET", "/", nil, nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.Header.Get("Authorization") != client.Authorization {
				t.Fatal("expected different Authorization here")
			}
		})

		t.Run("with accept", func(t *testing.T) {
			client := newAPIClient()
			client.Accept = "application/xml"
			req, err := client.newRequestWithJSONBody(
				context.Background(), "GET", "/", nil, []string{},
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.Header.Get("Accept") != "application/xml" {
				t.Fatal("expected different Accept here")
			}
		})

		t.Run("with custom host header", func(t *testing.T) {
			client := newAPIClient()
			client.Host = "www.x.org"
			req, err := client.newRequest(
				context.Background(), "GET", "/", nil, nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.Host != client.Host {
				t.Fatal("expected different req.Host here")
			}
		})

		t.Run("with user agent", func(t *testing.T) {
			client := newAPIClient()
			req, err := client.newRequest(
				context.Background(), "GET", "/", nil, nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if req.Header.Get("User-Agent") != userAgent {
				t.Fatal("expected different User-Agent here")
			}
		})
	})

	t.Run("doJSON", func(t *testing.T) {
		t.Run("do failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			client := newAPIClient()
			client.HTTPClient = &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return nil, expected
				},
			}
			err := client.doJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
		})

		t.Run("response is not successful (i.e., >= 400)", func(t *testing.T) {
			client := newAPIClient()
			client.HTTPClient = &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 401,
						Body:       io.NopCloser(strings.NewReader("{}")),
					}, nil
				},
			}
			err := client.doJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
			if !errors.Is(err, ErrRequestFailed) {
				t.Fatal("not the error we expected", err)
			}
		})

		t.Run("cannot read body", func(t *testing.T) {
			expected := errors.New("mocked error")
			client := newAPIClient()
			client.HTTPClient = &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(&mocks.Reader{
							MockRead: func(b []byte) (int, error) {
								return 0, expected
							},
						}),
					}, nil
				},
			}
			err := client.doJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
		})

		t.Run("response is not JSON", func(t *testing.T) {
			client := newAPIClient()
			client.HTTPClient = &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("[")),
					}, nil
				},
			}
			err := client.doJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
			if err == nil || err.Error() != "unexpected end of JSON input" {
				t.Fatal("not the error we expected")
			}
		})
	})

	t.Run("GetJSON", func(t *testing.T) {
		t.Run("successful case", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`["foo", "bar"]`))
				},
			))
			defer server.Close()
			ctx := context.Background()
			var result []string
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				Logger:     model.DiscardLogger,
			}).GetJSON(ctx, "/", &result)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != 2 || result[0] != "foo" || result[1] != "bar" {
				t.Fatal("invalid result", result)
			}
		})

		t.Run("failure case", func(t *testing.T) {
			var headers []string
			client := newAPIClient()
			client.BaseURL = "\t\t\t\t"
			err := client.GetJSON(context.Background(), "/", &headers)
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("not the error we expected")
			}
		})
	})

	t.Run("PostJSON", func(t *testing.T) {
		t.Run("successful case", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					var incoming []string
					data, err := netxlite.ReadAllContext(r.Context(), r.Body)
					if err != nil {
						w.WriteHeader(500)
						return
					}
					if err := json.Unmarshal(data, &incoming); err != nil {
						w.WriteHeader(500)
						return
					}
					w.Write(data)
				},
			))
			defer server.Close()
			ctx := context.Background()
			incoming := []string{"foo", "bar"}
			var result []string
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				Logger:     model.DiscardLogger,
			}).PostJSON(ctx, "/", incoming, &result)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(incoming, result); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("failure case", func(t *testing.T) {
			incoming := []string{"foo", "bar"}
			var result []string
			client := newAPIClient()
			client.BaseURL = "\t\t\t\t"
			err := client.PostJSON(context.Background(), "/", incoming, &result)
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("not the error we expected")
			}
		})
	})

	t.Run("FetchResource", func(t *testing.T) {
		t.Run("successful case", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("deadbeef"))
				},
			))
			defer server.Close()
			ctx := context.Background()
			data, err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				Logger:     model.DiscardLogger,
			}).FetchResource(ctx, "/")
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "deadbeef" {
				t.Fatal("invalid data")
			}
		})

		t.Run("failure case", func(t *testing.T) {
			client := newAPIClient()
			client.BaseURL = "\t\t\t\t"
			data, err := client.FetchResource(context.Background(), "/")
			if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
				t.Fatal("not the error we expected")
			}
			if data != nil {
				t.Fatal("unexpected data")
			}
		})
	})

	t.Run("we honour context", func(t *testing.T) {
		// It should suffice to check one of the public methods here
		client := newAPIClient()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // test should fail
		data, err := client.FetchResource(ctx, "/")
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
		if data != nil {
			t.Fatal("unexpected data")
		}
	})

	t.Run("body logging", func(t *testing.T) {
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
			ctx := context.Background()
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				LogBody:    true,
				Logger: &mocks.Logger{
					MockDebugf: func(format string, v ...interface{}) {
						logs <- fmt.Sprintf(format, v...)
					},
				},
			}).PostJSON(ctx, "/", input, &output)
			var found int
			close(logs)
			for entry := range logs {
				if strings.HasPrefix(entry, "httpx: request body: ") {
					found |= 1 << 0
					continue
				}
				if strings.HasPrefix(entry, "httpx: response body: ") {
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
			ctx := context.Background()
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				LogBody:    true,
				Logger: &mocks.Logger{
					MockDebugf: func(format string, v ...interface{}) {
						logs <- fmt.Sprintf(format, v...)
					},
				},
			}).PostJSON(ctx, "/", input, &output)
			var found int
			close(logs)
			for entry := range logs {
				if strings.HasPrefix(entry, "httpx: request body: ") {
					found |= 1 << 0
					continue
				}
				if strings.HasPrefix(entry, "httpx: response body: ") {
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
			ctx := context.Background()
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				LogBody:    false, // explicit initialization
				Logger: &mocks.Logger{
					MockDebugf: func(format string, v ...interface{}) {
						logs <- fmt.Sprintf(format, v...)
					},
				},
			}).PostJSON(ctx, "/", input, &output)
			var found int
			close(logs)
			for entry := range logs {
				if strings.HasPrefix(entry, "httpx: request body: ") {
					found |= 1 << 0
					continue
				}
				if strings.HasPrefix(entry, "httpx: response body: ") {
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
			ctx := context.Background()
			err := (&apiClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
				LogBody:    false, // explicit initialization
				Logger: &mocks.Logger{
					MockDebugf: func(format string, v ...interface{}) {
						logs <- fmt.Sprintf(format, v...)
					},
				},
			}).PostJSON(ctx, "/", input, &output)
			var found int
			close(logs)
			for entry := range logs {
				if strings.HasPrefix(entry, "httpx: request body: ") {
					found |= 1 << 0
					continue
				}
				if strings.HasPrefix(entry, "httpx: response body: ") {
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
	})
}
