package testingx

import (
	"io"
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSOverHTTPSHandler is an [http.Handler] implementing DNS-over-HTTPS.
type DNSOverHTTPSHandler struct {
	// Config is the MANDATORY config telling this DNS server which specific mappings
	// between domain names and IP addresses it knows.
	Config *netem.DNSConfig
}

var _ http.Handler = &DNSOverHTTPSHandler{}

// ServeHTTP implements [http.Handler].
func (p *DNSOverHTTPSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer p.handlePanic(w)
	rawQuery := runtimex.Try1(io.ReadAll(r.Body))
	rawResponse := runtimex.Try1(netem.DNSServerRoundTrip(p.Config, rawQuery))
	w.Header().Add("content-type", "application/dns-message")
	w.Write(rawResponse)
}

func (p *DNSOverHTTPSHandler) handlePanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		w.WriteHeader(500)
	}
}
