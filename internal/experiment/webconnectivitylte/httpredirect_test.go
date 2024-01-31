package webconnectivitylte

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPRedirectIsRedirect(t *testing.T) {
	type testcase struct {
		status int
		expect bool
	}

	cases := []testcase{{
		status: 100,
		expect: false,
	}, {
		status: 200,
		expect: false,
	}, {
		status: 300,
		expect: false,
	}, {
		status: 301,
		expect: true,
	}, {
		status: 302,
		expect: true,
	}, {
		status: 304,
		expect: false,
	}, {
		status: 305,
		expect: false,
	}, {
		status: 306,
		expect: false,
	}, {
		status: 307,
		expect: true,
	}, {
		status: 308,
		expect: true,
	}, {
		status: 309,
		expect: false,
	}, {
		status: 400,
		expect: false,
	}, {
		status: 500,
		expect: false,
	}}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d", tc.status), func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status}
			got := httpRedirectIsRedirect(resp)
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestHTTPValidateRedirect(t *testing.T) {
	type testcase struct {
		addReq   bool
		location string
		expect   error
	}

	cases := []testcase{{
		addReq:   false,
		location: "/en-US/index.html",
		expect:   errHTTPValidateRedirectMissingRequest,
	}, {
		addReq:   true,
		location: "", // explicitly empty
		expect:   http.ErrNoLocation,
	}, {
		addReq:   true,
		location: "http://",
		expect:   errors.New(netxlite.FailureHTTPInvalidRedirectLocationHost),
	}, {
		addReq:   true,
		location: "https://",
		expect:   errors.New(netxlite.FailureHTTPInvalidRedirectLocationHost),
	}, {
		addReq:   true,
		location: "/en-US/index.html",
		expect:   nil,
	}, {
		addReq:   true,
		location: "https://web01.example.com/",
		expect:   nil,
	}}

	for _, tc := range cases {
		t.Run(tc.location, func(t *testing.T) {
			resp := &http.Response{Header: http.Header{}}
			resp.Header.Set("Location", tc.location)
			if tc.addReq {
				resp.Request = &http.Request{URL: &url.URL{
					Scheme: "https",
					Host:   "www.example.com",
					Path:   "/",
				}}
			}

			got := httpValidateRedirect(resp)

			switch {
			case tc.expect == nil && got == nil:
				// all good

			case tc.expect == nil && got != nil:
				t.Fatal("expected", tc.expect, "got", got)

			case tc.expect != nil && got == nil:
				t.Fatal("expected", tc.expect, "got", got)

			case tc.expect != nil && got != nil:
				if diff := cmp.Diff(tc.expect.Error(), got.Error()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
