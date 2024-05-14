package testingx

import (
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSOverHTTPSHandler is an [http.Handler] implementing DNS-over-HTTPS.
type DNSOverHTTPSHandler struct {
	// RoundTripper is the MANDATORY round tripper to use.
	RoundTripper DNSRoundTripper
}

var _ http.Handler = &DNSOverHTTPSHandler{}

// ServeHTTP implements [http.Handler].
func (p *DNSOverHTTPSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer p.handlePanic(w)
	rawQuery := runtimex.Try1(io.ReadAll(r.Body))
	rawResponse := runtimex.Try1(p.RoundTripper.RoundTrip(r.Context(), rawQuery))
	w.Header().Add("content-type", "application/dns-message")
	_, _ = w.Write(rawResponse)
}

func (p *DNSOverHTTPSHandler) handlePanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
