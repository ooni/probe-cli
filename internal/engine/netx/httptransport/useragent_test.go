package httptransport_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
)

func TestUserAgentWithDefault(t *testing.T) {
	txp := httptransport.UserAgentTransport{
		RoundTripper: httptransport.FakeTransport{
			Resp: &http.Response{StatusCode: 200},
		},
	}
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "www.google.com",
		Path:   "/",
	}}
	req.Header = http.Header{}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Request.Header.Get("User-Agent") != "miniooni/0.1.0-dev" {
		t.Fatal("not the User-Agent we expected")
	}
}

func TestUserAgentWithExplicitValue(t *testing.T) {
	txp := httptransport.UserAgentTransport{
		RoundTripper: httptransport.FakeTransport{
			Resp: &http.Response{StatusCode: 200},
		},
	}
	req := &http.Request{URL: &url.URL{
		Scheme: "https",
		Host:   "www.google.com",
		Path:   "/",
	}}
	req.Header = http.Header{"User-Agent": []string{"antani-client/0.1.1"}}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Request.Header.Get("User-Agent") != "antani-client/0.1.1" {
		t.Fatal("not the User-Agent we expected")
	}
}
