package nwcth

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
)

var nexturlch = make(chan *nwebconnectivity.NextLocationInfo)

func TestHTTPDoWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	resp := HTTPDo(ctx, &HTTPConfig{
		Client:            http.DefaultClient,
		Headers:           nil,
		MaxAcceptableBody: 1 << 24,
		URL:               "http://[::1]aaaa",
	}, nexturlch)
	if resp.Failure == nil || !strings.HasSuffix(*resp.Failure, `invalid port "aaaa" after host`) {
		t.Fatal("not the failure we expected")
	}
}

func TestHTTPDoWithHTTP3(t *testing.T) {
	ctx := context.Background()
	resp := HTTPDo(ctx, &HTTPConfig{
		Client: &http.Client{
			Transport: &http3.RoundTripper{},
		},
		Headers:           nil,
		MaxAcceptableBody: 1 << 24,
		URL:               "https://www.google.com",
	}, nexturlch)
	if resp.Failure != nil {
		t.Fatal(resp.Failure)
	}
}
