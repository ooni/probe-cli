package oohelperd

import (
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/browserhttp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestUseBrowserClient(t *testing.T) {
	h := NewHandler(model.DiscardLogger, &netxlite.Netx{})
	browser := http.Header{}
	browserhttp.SetHeaders(browser, false)
	older := map[string][]string{"User-Agent": {model.HTTPHeaderUserAgent}}

	if !h.useBrowserClient(browser) {
		t.Error("a browser probe doesn't get the browser client")
	}
	if h.useBrowserClient(older) {
		t.Error("an older probe gets the browser client")
	}
	h.newBrowserHTTPClient = nil
	if h.useBrowserClient(browser) {
		t.Error("used a browser client we don't have")
	}
}
