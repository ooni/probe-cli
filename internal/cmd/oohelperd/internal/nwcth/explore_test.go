package nwcth

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestExploreSuccess(t *testing.T) {
	u, err := url.Parse("https://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts, err := Explore(u)
	if err != nil {
		t.Fatal("unexpected error")
	}
	if len(rts) != 1 {
		t.Fatal("unexpected number of roundtrips")
	}
}

func TestExploreFailure(t *testing.T) {
	u, err := url.Parse("https://example.example")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts, err := Explore(u)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if rts != nil {
		t.Fatal("rts should be nil")
	}
}

func TestExploreSuccessWithH3(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts, err := Explore(u)
	if err != nil {
		t.Fatal("unexpected error")
	}
	if len(rts) != 2 {
		t.Fatal("unexpected number of roundtrips")
	}
	if rts[0].proto != "https" {
		t.Fatal("unexpected protocol")
	}
	if rts[1].proto != "h3" {
		t.Fatal("unexpected protocol")
	}
}

func TestGetSuccess(t *testing.T) {
	u, err := url.Parse("https://example.com")
	resp, err := get(u)
	if err != nil {
		t.Fatal("unexpected error")
	}
	if resp == nil {
		t.Fatal("unexpected nil response")
	}
	buf := make([]byte, 100)
	if n, _ := resp.Body.Read(buf); n != 0 {
		t.Fatal("expected response body tom be closed")
	}

}

func TestGetFailure(t *testing.T) {
	u, err := url.Parse("https://example.example")
	resp, err := get(u)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("response should be nil")
	}
}

func TestGetH3Success(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	h3u := &h3URL{URL: u, proto: "h3"}
	resp, err := getH3(h3u)
	if err != nil {
		t.Fatal("unexpected error")
	}
	if resp == nil {
		t.Fatal("unexpected nil response")
	}
	buf := make([]byte, 100)
	if n, _ := resp.Body.Read(buf); n != 0 {
		t.Fatal("expected response body tom be closed")
	}

}

func TestGetH3Failure(t *testing.T) {
	u, err := url.Parse("https://www.google.google")
	h3u := &h3URL{URL: u, proto: "h3"}
	resp, err := getH3(h3u)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("response should be nil")
	}
}

func TestRearrange(t *testing.T) {
	u, err := url.Parse("https://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	resp := &http.Response{
		// the ProtoMajor field identifies the request/response structs and indicates the correct order
		ProtoMajor: 2,
		Request: &http.Request{
			ProtoMajor: 2,
			URL:        u,
			Response: &http.Response{
				ProtoMajor: 1,
				Request: &http.Request{
					ProtoMajor: 1,
					URL:        u,
					Response: &http.Response{
						ProtoMajor: 0,
						Request: &http.Request{
							ProtoMajor: 0,
							URL:        u,
						},
					},
				},
			},
		},
	}
	proto := "expected"
	rts := rearrange(resp, &proto)
	expectedIndex := 0
	for _, rt := range rts {
		if rt.Request == nil || rt.Response == nil {
			t.Fatal("unexpected nil value")
		}
		if rt.Request.ProtoMajor != expectedIndex {
			t.Fatal("unexpected order")
		}
		if rt.Response.ProtoMajor != expectedIndex {
			t.Fatal("unexpected order")
		}
		if rt.proto != proto {
			t.Fatal("unexpected protocol")
		}
		expectedIndex += 1
	}

}
