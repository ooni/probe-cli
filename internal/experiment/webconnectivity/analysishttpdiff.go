package webconnectivity

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// analysisHTTPDiff computes the HTTP diff between the final request-response
// observed by the probe and the TH's result. The caller is responsible of passing
// us a valid probe observation and a valid TH observation.
func (tk *TestKeys) analysisHTTPDiff(
	probe *model.ArchivalHTTPRequestResult, th *webconnectivity.ControlHTTPRequestResult) {
	// TODO
}
