package nwcth

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func safeURLParse(s string) *url.URL {
	url, err := url.Parse(s)
	runtimex.PanicOnError(err, "url.Parse failed")
	return url
}

func TestHTTPNoH3Transport(t *testing.T) {
	ctx := context.Background()
	url := safeURLParse("https://ooni.org")
	transport := http.DefaultTransport
	ctrl, nextlocation := HTTPDo(ctx, &HTTPConfig{
		Jar:       nil,
		Headers:   nil,
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
	}
	if nextlocation != nil {
		t.Fatal("unexpected next location")
	}
	h3Support := discoverH3Server(ctrl, url)
	if h3Support != "" {
		t.Fatal("not the h3 support value we expected")
	}
}

func TestHTTPDoWithH3Transport(t *testing.T) {
	ctx := context.Background()
	url := safeURLParse("https://www.google.com")
	transport := http.DefaultTransport
	ctrl, nextlocation := HTTPDo(ctx, &HTTPConfig{
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
	}
	if nextlocation != nil {
		t.Fatal("unexpected next location")
	}
	h3Support := discoverH3Server(ctrl, url)
	if h3Support != "h3" {
		t.Fatal("not the h3 support value we expected")
	}

	transport = &http3.RoundTripper{
		TLSClientConfig: &tls.Config{ServerName: url.Hostname()},
		QuicConfig:      &quic.Config{},
	}
	ctrl, nextlocation = HTTPDo(ctx, &HTTPConfig{
		Jar:       nil,
		Headers:   nil,
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
	}
	if nextlocation != nil {
		t.Fatal("unexpected next location")
	}
}

type H3ServerSupport struct {
	url        string
	expectedh3 string
}

func TestDiscoverH3Server(t *testing.T) {
	tests := []H3ServerSupport{
		{
			url:        "https://www.google.com",
			expectedh3: "h3",
		},
		{
			url:        "https://www.facebook.com",
			expectedh3: "h3-29",
		},
		{
			url:        "https://ooni.org",
			expectedh3: "",
		},
	}
	transport := http.DefaultTransport
	for _, testcase := range tests {
		URL := safeURLParse(testcase.url)
		ctrl, err := HTTPDo(context.Background(), &HTTPConfig{
			Transport: transport,
			URL:       URL,
		})
		if err != nil {
			t.Fatal("unexpected error")
		}
		if discoverH3Server(ctrl, URL) != testcase.expectedh3 {
			t.Fatal("unexpected h3 support string")
		}
	}
	URL := safeURLParse("https://www.google.com")
	if discoverH3Server(nil, URL) != "" {
		t.Fatal("unexpected h3 support string")
	}
}

func TestHTTPDoWithHTTPTransportFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	URL := safeURLParse("http://www.x.org")
	ctrl, nextlocation := HTTPDo(ctx, &HTTPConfig{
		Transport: FakeTransport{
			Err: expected,
		},
		Headers: nil,
		URL:     URL,
	})
	if ctrl.Failure == nil || !strings.HasSuffix(*ctrl.Failure, "mocked error") {
		t.Fatal("not the error we expected")
	}
	if nextlocation != nil {
		t.Fatal("unexpected next location")
	}
}
