package nwcth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

const requestnoredirect = `{
	"http_request": "https://ooni.org",
	"http_request_headers": {
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
	"tcp_connect": [
	  "104.198.14.52:443"
	]
}`

const requestsimpleredirect = `{
	"http_request": "https://www.ooni.org",
	"http_request_headers": {
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
	"tcp_connect": [
	  "18.192.76.182:443"
	]
}`

const requestmultipleredirect = `{
	"http_request": "http://яндекс.рф",
	"http_request_headers": {
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
	"tcp_connect": [
	  "77.88.55.70:443"
	]
}`

const requestwithquic = `{
	"http_request": "https://www.google.com",
	"http_request_headers": {
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
	"tcp_connect": [
	  "142.250.74.196:443"
	]
}`

const requestWithoutDomainName = `{
	"http_request": "https://8.8.8.8",
	"http_request_headers": {
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
	"tcp_connect": [
	  "8.8.8.8:443"
	]
}`

func TestWorkingAsIntended(t *testing.T) {
	handler := Handler{}
	srv := httptest.NewServer(handler)
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
		reqBody:        `{"http_request": "http://[::1]aaaa"}`,
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
		reqBody:         requestsimpleredirect,
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name:            "request triggering multiple redirects",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         requestmultipleredirect,
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

func TestHandlerWithRequestBodyReadingError(t *testing.T) {
	expected := errors.New("mocked error")
	handler := Handler{}
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
