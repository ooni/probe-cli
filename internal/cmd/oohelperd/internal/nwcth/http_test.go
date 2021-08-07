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
	quicURL, err := getH3Location(ctrl, url)
	if err == nil {
		t.Fatal("exptected error an error here")
	}
	if quicURL != nil {
		t.Fatal("unexpected quic URL")
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
	quicURL, err := getH3Location(ctrl, url)
	if err != nil {
		t.Fatal("unexpected error")
	}
	if quicURL == nil {
		t.Fatal("expected an h3 alt-svc entry here")
	}
	if quicURL.Scheme != "h3" {
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
	h3tests := []H3ServerSupport{
		{
			url:        "https://www.google.com",
			expectedh3: "h3",
		},
		{
			url:        "https://www.facebook.com",
			expectedh3: "h3-29",
		},
	}
	transport := http.DefaultTransport
	for _, testcase := range h3tests {
		URL := safeURLParse(testcase.url)
		ctrl, err := HTTPDo(context.Background(), &HTTPConfig{
			Transport: transport,
			URL:       URL,
		})
		if err != nil {
			t.Fatal("unexpected error")
		}
		quicURL, qerr := getH3Location(ctrl, URL)
		if qerr != nil {
			t.Fatal("unexpected error")
		}
		if quicURL == nil {
			t.Fatal("expected an h3 alt-svc entry here")
		}
		if quicURL.Scheme != testcase.expectedh3 {
			t.Fatal("unexpected h3 support string")
		}
	}
	URL := safeURLParse("https://ooni.org")
	quicURL, err := getH3Location(nil, URL)
	if err == nil {
		t.Fatal("exptected an error here")
	}
	if quicURL != nil {
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
