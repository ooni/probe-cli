package websteps

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	errorsxlegacy "github.com/ooni/probe-cli/v3/internal/engine/legacy/errorsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	thAddr string, resourcePath string, creq CtrlRequest) (out CtrlResponse, err error) {
	clnt := &httpx.APIClient{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
	}
	// make sure error is wrapped
	err = errorsxlegacy.SafeErrWrapperBuilder{
		Error:     clnt.PostJSON(ctx, resourcePath, creq, &out),
		Operation: netxlite.TopLevelOperation,
	}.MaybeBuild()
	return
}
