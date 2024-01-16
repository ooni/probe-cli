package webconnectivitylte

//
// HTTP core analysis
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// analysisHTTPToplevel is the toplevel analysis function for HTTP results.
//
// This function's job is to determine whether there were unexpected TLS
// handshake results (compared to what the TH observed), or unexpected
// failures during HTTP round trips (using the TH as benchmark), or whether
// the obtained body differs from the one obtained by the TH.
//
// This results in possibly setting these XBlockingFlags:
//
// - AnalysisBlockingFlagHTTPBlocking
//
// - AnalysisBlockingFlagHTTPDiff
//
// In websteps fashion, we don't stop at the first failure, rather we
// process all the available data and evaluate all possible errors.
func (tk *TestKeys) analysisHTTPToplevel(logger model.Logger) {
	// if we don't have any request to check, there's not much more we
	// can actually do here, so let's just return.
	if len(tk.Requests) <= 0 {
		return
	}
	// TODO(https://github.com/ooni/probe/issues/2641): this code is wrong
	// with redirects because LTE only creates an HTTP request when it reaches
	// the HTTP stage, so a previous redirect that is successful would cause
	// this code to say we're good on the HTTP front, while we're not.
	finalRequest := tk.Requests[0]
	if finalRequest.Failure != nil {
		tk.HTTPExperimentFailure = optional.Some(*finalRequest.Failure)
	}

	// don't perform any futher analysis without TH data
	if tk.Control == nil || tk.ControlRequest == nil {
		return
	}
	ctrl := tk.Control.HTTPRequest

	// don't perform any analysis if the TH's HTTP measurement failed because
	// performing more precise mapping is a job for the pipeline.
	if ctrl.Failure != nil {
		return
	}

	// flag cases of known HTTP failures
	if failure := finalRequest.Failure; failure != nil {
		switch *failure {
		case netxlite.FailureConnectionReset,
			netxlite.FailureGenericTimeoutError,
			netxlite.FailureEOFError:
			tk.BlockingFlags |= AnalysisBlockingFlagHTTPBlocking
			logger.Warnf(
				"HTTP: unexpected failure %s for %s (see #%d)",
				*failure,
				finalRequest.Address,
				finalRequest.TransactionID,
			)
		default:
			// leave this case for ooni/pipeline
		}
		return
	}

	// fallback to the HTTP diff algo.
	tk.analysisHTTPDiff(logger, finalRequest, &ctrl)
}
