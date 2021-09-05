package websteps

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

const requestnoredirect = `{
	"url": "https://ooni.org",
	"headers": {
	  "Accept": [
		"*/*"
	  ],
	  "Accept-Language": [
		"en-US;q=0.8,en;q=0.5"
	  ],
	  "User-Agent": [
		"Mozilla/5.0"
	  ]
	},
	"addrs": [
	  "104.198.14.52:443"
	]
}`

const requestredirect = `{
	"url": "https://www.ooni.org",
	"headers": {
	  "Accept": [
		"*/*"
	  ],
	  "Accept-Language": [
		"en-US;q=0.8,en;q=0.5"
	  ],
	  "User-Agent": [
		"Mozilla/5.0"
	  ]
	},
	"addrs": [
	  "18.192.76.182:443"
	]
}`

const requestIPaddressinput = `{
	"url": "https://172.217.168.4",
	"headers": {
	  "Accept": [
		"*/*"
	  ],
	  "Accept-Language": [
		"en-US;q=0.8,en;q=0.5"
	  ],
	  "User-Agent": [
		"Mozilla/5.0"
	  ]
	},
	"addrs": [
	  "172.217.168.4:443"
	]
}`

const requestwithquic = `{
	"url": "https://www.google.com",
	"headers": {
	  "Accept": [
		"*/*"
	  ],
	  "Accept-Language": [
		"en-US;q=0.8,en;q=0.5"
	  ],
	  "User-Agent": [
		"Mozilla/5.0"
	  ]
	},
	"addrs": [
	  "142.250.74.196:443"
	]
}`

const requestWithoutDomainName = `{
	"url": "https://8.8.8.8",
	"headers": {
	  "Accept": [
		"*/*"
	  ],
	  "Accept-Language": [
		"en-US;q=0.8,en;q=0.5"
	  ],
	  "User-Agent": [
		"Mozilla/5.0"
	  ]
	},
	"addrs": [
	  "8.8.8.8:443"
	]
}`

func TestWorkingAsIntended(t *testing.T) {
	handler := Handler{Config: &Config{}}
	srv := httptest.NewServer(&handler)
	defer srv.Close()
	type expectationSpec struct {
		name            string
		reqMethod       string
		reqContentType  string
		reqBody         string
		respStatusCode  int
		respContentType string
		parseBody       bool
	}
	expectations := []expectationSpec{{
		name:           "check for invalid method",
		reqMethod:      "GET",
		respStatusCode: 400,
	}, {
		name:           "check for invalid request body",
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqBody:        "{",
		respStatusCode: 400,
	}, {
		name:           "with measurement failure",
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqBody:        `{"url": "http://[::1]aaaa"}`,
		respStatusCode: 400,
	}, {
		name:            "request without redirect or H3 follow-up request",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestnoredirect,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name:            "request triggering one redirect, without H3 follow-up request",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestredirect,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name:            "request with an IP address as input",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestIPaddressinput,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name:            "request triggering H3 follow-up request, without redirect",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestwithquic,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name:            "when there's no domain name in the request",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestWithoutDomainName,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}}
	for _, expect := range expectations {
		t.Run(expect.name, func(t *testing.T) {
			body := strings.NewReader(expect.reqBody)
			req, err := http.NewRequest(expect.reqMethod, srv.URL, body)
			if err != nil {
				t.Fatalf("%s: %+v", expect.name, err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("%s: %+v", expect.name, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != expect.respStatusCode {
				t.Fatalf("unexpected status code: %+v", resp.StatusCode)
			}
			data, err := iox.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if !expect.parseBody {
				return
			}
			var v interface{}
			if err := json.Unmarshal(data, &v); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHandlerWithInternalServerError(t *testing.T) {
	handler := Handler{Config: &Config{explorer: &MockExplorer{}}}
	srv := httptest.NewServer(&handler)
	defer srv.Close()
	body := strings.NewReader(`{"url": "https://example.com"}`)
	req, err := http.NewRequest("POST", srv.URL, body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Fatalf("unexpected status code: %+v", resp.StatusCode)
	}
	_, err = iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandlerWithRequestBodyReadingError(t *testing.T) {
	expected := errors.New("mocked error")
	handler := Handler{Config: &Config{}}
	rw := NewFakeResponseWriter()
	req := &http.Request{
		Method: "POST",
		Header: map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {"2048"},
		},
		Body: &FakeBody{Err: expected},
	}
	handler.ServeHTTP(rw, req)
	if rw.StatusCode != 400 {
		t.Fatal("unexpected status code")
	}
}

type FakeBody struct {
	Err error
}

func (fb FakeBody) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	return 0, fb.Err
}

func (fb FakeBody) Close() error {
	return nil
}

type FakeResponseWriter struct {
	Body       [][]byte
	HeaderMap  http.Header
	StatusCode int
}

func NewFakeResponseWriter() *FakeResponseWriter {
	return &FakeResponseWriter{HeaderMap: make(http.Header)}
}

func (frw *FakeResponseWriter) Header() http.Header {
	return frw.HeaderMap
}

func (frw *FakeResponseWriter) Write(b []byte) (int, error) {
	frw.Body = append(frw.Body, b)
	return len(b), nil
}

func (frw *FakeResponseWriter) WriteHeader(statusCode int) {
	frw.StatusCode = statusCode
}
