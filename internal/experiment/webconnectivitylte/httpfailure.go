package webconnectivitylte

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TODO(bassosimone): document this func
func newArchivalHTTPRequestResultWithError(trace *measurexlite.Trace, network, address, alpn string,
	req *http.Request, err error) *model.ArchivalHTTPRequestResult {
	duration := trace.TimeSince(trace.ZeroTime())
	return measurexlite.NewArchivalHTTPRequestResult(
		trace.Index(),
		duration,
		network,
		address,
		alpn,
		network, // TODO(bassosimone): get rid of this duplicate field?
		req,
		nil,
		0,
		nil,
		err,
		duration,
	)
}
