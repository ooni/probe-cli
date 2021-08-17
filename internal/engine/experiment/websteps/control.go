package websteps

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// CtrlRequest is the request sent by the probe
type CtrlRequest struct {
	HTTPRequest        string              `json:"url"`
	HTTPRequestHeaders map[string][]string `json:"headers"`
	Addrs              []string            `json:"addrs"`
}

// ControlResponse is the response from the control service.
type ControlResponse struct {
	URLMeasurements []*URLMeasurement `json:"urls"`
}

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, creq CtrlRequest) (out ControlResponse, err error) {
	clnt := httpx.Client{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
	}
	// make sure error is wrapped
	err = errorsx.SafeErrWrapperBuilder{
		Error:     clnt.PostJSON(ctx, "/", creq, &out),
		Operation: errorsx.TopLevelOperation,
	}.MaybeBuild()
	return
}
