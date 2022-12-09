package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const simplerequest = `{
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

func TestHandlerWorkingAsIntended(t *testing.T) {
	// TODO(bassosimone): we should factor the function that constructs a
	// handler inside of main.go and reuse it here. As much as this is just
	// an integration test, it seems backwards to initialize the handler
	// differently than we use it in production (and I don't see a reason
	// why we should be doing that).
	handler := &handler{
		BaseLogger:        model.DiscardLogger,
		Indexer:           &atomicx.Int64{},
		MaxAcceptableBody: 1 << 24,
		NewHTTPClient: func(model.Logger) model.HTTPClient {
			return http.DefaultClient
		},
		NewHTTP3Client: func(logger model.Logger) model.HTTPClient {
			return netxlite.NewHTTP3ClientWithResolver(
				model.DiscardLogger, newResolver(model.DiscardLogger))
		},
		NewDialer: func(model.Logger) model.Dialer {
			return netxlite.NewDialerWithoutResolver(model.DiscardLogger)
		},
		NewQUICDialer: func(logger model.Logger) model.QUICDialer {
			return netxlite.NewQUICDialerWithoutResolver(
				netxlite.NewQUICListener(),
				model.DiscardLogger,
			)
		},
		NewResolver: func(model.Logger) model.Resolver {
			return netxlite.NewUnwrappedStdlibResolver()
		},
		NewTLSHandshaker: func(model.Logger) model.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(model.DiscardLogger)
		},
	}
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
		name:            "check for invalid method",
		reqMethod:       "GET",
		reqContentType:  "",
		reqBody:         "",
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "check for invalid content-type",
		reqMethod:       "POST",
		reqContentType:  "",
		reqBody:         "",
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "check for invalid request body",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         "{",
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "with measurement failure",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         `{"http_request": "http://[::1]aaaa"}`,
		respStatusCode:  400,
		respContentType: "",
		parseBody:       false,
	}, {
		name:            "with reasonably good request",
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqBody:         simplerequest,
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
			if expect.reqContentType != "" {
				req.Header.Add("content-type", expect.reqContentType)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("%s: %+v", expect.name, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != expect.respStatusCode {
				t.Fatalf("unexpected status code: %+v", resp.StatusCode)
			}
			if v := resp.Header.Get("content-type"); v != expect.respContentType {
				t.Fatalf("unexpected content-type: %s", v)
			}
			data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
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
	handler := handler{MaxAcceptableBody: 1 << 24}
	var statusCode int
	headers := http.Header{}
	rw := &mocks.HTTPResponseWriter{
		MockWriteHeader: func(code int) {
			statusCode = code
		},
		MockHeader: func() http.Header {
			return headers
		},
	}
	req := &http.Request{
		Method: "POST",
		Header: map[string][]string{
			"Content-Type":   {"application/json"},
			"Content-Length": {"2048"},
		},
		Body: io.NopCloser(&mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				return 0, expected
			},
		}),
	}
	handler.ServeHTTP(rw, req)
	if statusCode != 400 {
		t.Fatal("unexpected status code")
	}
}
