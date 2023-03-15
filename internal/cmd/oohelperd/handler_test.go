package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

// simpleRequestForHandler is a simple request for the [handler].
const simpleRequestForHandler = `{
	"http_request": "https://dns.google",
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

// requestWithDomainName is input for testing the [handler].
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
	],
	"x_quic_enabled": true
}`

// TestHandlerWorkingAsIntended is an unit test exercising
// several code paths inside the [handler].
func TestHandlerWorkingAsIntended(t *testing.T) {

	// expectationSpec describes our expectations
	type expectationSpec struct {
		// name is the name of the subtest
		name string

		// reqMethod is the method for the HTTP request
		reqMethod string

		// reqContentType is the content-type for the HTTP request
		reqContentType string

		// measureFn optionally allows overriding the default
		// value of the handler.Measure function
		measureFn func(
			ctx context.Context, config *handler, creq *model.THRequest) (*model.THResponse, error)

		// reqBody is the request body to use
		reqBody io.Reader

		// respStatusCode is the expected response status code
		respStatusCode int

		// respContentType is the expected content-type
		respContentType string

		// parseBody indicates whether this test should attempt
		// to parse the response body
		parseBody bool
	}

	expectations := []expectationSpec{{
		name:            "check for invalid method",
		reqMethod:       "GET",
		reqContentType:  "",
		reqBody:         strings.NewReader(""),
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:           "check for error reading request body",
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqBody: &mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				return 0, errors.New("connection reset by peer")
			},
		},
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "check for invalid request body",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         strings.NewReader("{"),
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "with measurement failure",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         strings.NewReader(`{"http_request": "http://[::1]aaaa"}`),
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name: "with reasonably good request",
		measureFn: func(ctx context.Context, config *handler, creq *model.THRequest) (*model.THResponse, error) {
			cresp := &model.THResponse{}
			return cresp, nil
		},
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         strings.NewReader(simpleRequestForHandler),
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}, {
		name: "with request that does not contain a domain name",
		// TODO(bassosimone): this subtest is still an integration test because
		// it tests part of measure.go. We should create unit tests for measure.go
		// and remove this test from this file.
		measureFn:       measure,
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         strings.NewReader(requestWithoutDomainName),
		respStatusCode:  200,
		respContentType: "application/json",
		parseBody:       true,
	}}

	for _, expect := range expectations {
		t.Run(expect.name, func(t *testing.T) {
			// create handler and possibly override .Measure
			handler := newHandler()
			if expect.measureFn != nil {
				handler.Measure = expect.measureFn
			}

			// create request
			req, err := http.NewRequestWithContext(
				context.Background(),
				expect.reqMethod,
				"http://127.0.0.1:8080/",
				expect.reqBody,
			)
			if err != nil {
				t.Fatalf("%s: %+v", expect.name, err)
			}
			if expect.reqContentType != "" {
				req.Header.Add("content-type", expect.reqContentType)
			}

			// create response writer
			var (
				respBody       = []byte{}
				header         = http.Header{}
				statusCode int = 200
			)
			rw := &mocks.HTTPResponseWriter{
				MockHeader: func() http.Header {
					return header
				},
				MockWrite: func(b []byte) (int, error) {
					respBody = append(respBody, b...)
					return len(b), nil
				},
				MockWriteHeader: func(code int) {
					statusCode = code
				},
			}

			// perform round trip
			handler.ServeHTTP(rw, req)

			// process response
			if statusCode != expect.respStatusCode {
				t.Fatalf("unexpected status code: %+v", statusCode)
			}
			if v := header.Get("content-type"); v != expect.respContentType {
				t.Fatalf("unexpected content-type: %s", v)
			}
			if !expect.parseBody {
				return
			}
			var v interface{}
			if err := json.Unmarshal(respBody, &v); err != nil {
				t.Fatal(err)
			}
		})
	}
}
