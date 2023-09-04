package netemx

import (
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// GeoIPHandlerFactoryUbuntu is a [QAEnvHTTPHandlerFactory] for [testingx.GeoIPHandlerUbuntu].
type GeoIPHandlerFactoryUbuntu struct {
	ProbeIP string
}

var _ HTTPHandlerFactory = &GeoIPHandlerFactoryUbuntu{}

// NewHandler implements QAEnvHTTPHandlerFactory.
func (f *GeoIPHandlerFactoryUbuntu) NewHandler(_ *netem.UNetStack) http.Handler {
	return &testingx.GeoIPHandlerUbuntu{ProbeIP: f.ProbeIP}
}
