package websteps

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestGetH3URLInvalidURL(t *testing.T) {
	resp := &http.Response{
		Request: &http.Request{},
	}
	h3URL, err := getH3URL(resp)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if h3URL != nil {
		t.Fatal("h3URL should be nil")
	}

}

func TestParseUnsupportedScheme(t *testing.T) {
	URL, err := url.Parse("h3://google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	parsed, err := parseAltSvc(nil, URL)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != ErrUnsupportedScheme {
		t.Fatal("unexpected error type")
	}
	if parsed != nil {
		t.Fatal("h3URL should be nil")
	}

}
