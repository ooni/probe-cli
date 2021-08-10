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
	ctrl := HTTPDo(ctx, &HTTPConfig{
		Jar:       nil,
		Headers:   nil,
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
	}
}

func TestHTTPDoWithH3Transport(t *testing.T) {
	ctx := context.Background()
	url := safeURLParse("https://www.google.com")
	transport := http.DefaultTransport
	ctrl := HTTPDo(ctx, &HTTPConfig{
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
	}

	transport = &http3.RoundTripper{
		TLSClientConfig: &tls.Config{ServerName: url.Hostname()},
		QuicConfig:      &quic.Config{},
	}
	ctrl = HTTPDo(ctx, &HTTPConfig{
		Jar:       nil,
		Headers:   nil,
		Transport: transport,
		URL:       url,
	})
	if ctrl.Failure != nil {
		t.Fatal("unexpected failure")
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
		ctrl := HTTPDo(context.Background(), &HTTPConfig{
			Transport: transport,
			URL:       URL,
		})
		if ctrl == nil {
			t.Fatal("unexpected nil value")
		}
	}
}

func TestHTTPDoWithHTTPTransportFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	URL := safeURLParse("http://www.x.org")
	ctrl := HTTPDo(ctx, &HTTPConfig{
		Transport: FakeTransport{
			Err: expected,
		},
		Headers: nil,
		URL:     URL,
	})
	if ctrl.Failure == nil || !strings.HasSuffix(*ctrl.Failure, "mocked error") {
		t.Fatal("not the error we expected")
	}
}
