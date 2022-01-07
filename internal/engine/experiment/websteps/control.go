package websteps

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, resourcePath string, creq CtrlRequest) (out CtrlResponse, err error) {
	clnt := &httpx.APIClientTemplate{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
	}
	// make sure error is wrapped
	err = clnt.WithBodyLogging().Build().PostJSON(ctx, resourcePath, creq, &out)
	if err != nil {
		err = netxlite.NewTopLevelGenericErrWrapper(err)
	}
	return
}
