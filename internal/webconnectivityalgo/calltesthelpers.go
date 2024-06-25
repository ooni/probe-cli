package webconnectivityalgo

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CallWebConnectivityTestHelper invokes the Web Connectivity test helper with the
// given request object, the given list of available test helpers, and the given session.
//
// If the list of test helpers is empty this function immediately returns nil, zero,
// and the [model.ErrNoAvailableTestHelpers] error to the caller.
//
// In case of any other failure, this function returns nil, zero, and an error.
//
// On success, it returns the response, the used TH index, and nil.
//
// Note that the returned error won't be wrapped, so you need to wrap it yourself.
func CallWebConnectivityTestHelper(ctx context.Context, creq *model.THRequest,
	testhelpers []model.OOAPIService, sess model.ExperimentSession) (*model.THResponse, int, error) {
	// handle the case where there are no available web connectivity test helpers
	if len(testhelpers) <= 0 {
		return nil, 0, model.ErrNoAvailableTestHelpers
	}

	// create overlapped state for performing overlapped HTTP calls
	overlapped := httpclientx.NewOverlappedPostJSON[*model.THRequest, *model.THResponse](
		creq, &httpclientx.Config{
			Authorization: "", // not needed
			Client:        sess.DefaultHTTPClient(),
			Logger:        sess.Logger(),
			UserAgent:     sess.UserAgent(),
		},
	)

	// perform the overlapped HTTP API calls
	cresp, idx, err := overlapped.Run(ctx, httpclientx.NewBaseURLsFromModelOOAPIServices(testhelpers...)...)

	// handle the case where all test helpers failed
	if err != nil {
		return nil, 0, err
	}

	// apply some sanity checks to the results
	runtimex.Assert(idx >= 0 && idx < len(testhelpers), "idx out of bounds")
	runtimex.Assert(cresp != nil, "out is nil")

	// return the results to the web connectivity caller
	return cresp, idx, nil
}
