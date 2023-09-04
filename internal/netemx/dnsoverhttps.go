package netemx

import (
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// DNSOverHTTPSHandlerFactory is a [QAEnvHTTPHandlerFactory] for [testingx.GeoIPHandlerUbuntu].
type DNSOverHTTPSHandlerFactory struct {
	Config *netem.DNSConfig
}

var _ HTTPHandlerFactory = &DNSOverHTTPSHandlerFactory{}

// NewHandler implements QAEnvHTTPHandlerFactory.
func (f *DNSOverHTTPSHandlerFactory) NewHandler(_ *netem.UNetStack) http.Handler {
	return &testingx.DNSOverHTTPSHandler{Config: f.Config}
}
