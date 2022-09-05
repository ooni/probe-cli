package webconnectivity

//
// HTTP core analysis
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
// - analysisFlagHTTPBlocking
//
// - analysisFlagHTTPDiff
//
// In websteps fashion, we don't stop at the first failure, rather we
// process all the available data and evaluate all possible errors.
func (tk *TestKeys) analysisHTTPToplevel(logger model.Logger) {
	// if we don't have any request to check, there's not much more we
	// can actually do here, so let's just return.
	if len(tk.Requests) <= 0 {
		return
	}
	finalRequest := tk.Requests[0]
	tk.HTTPExperimentFailure = finalRequest.Failure

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
			tk.BlockingFlags |= analysisFlagHTTPBlocking
			logger.Warnf(
				"HTTP: endpoint %s is blocked (see #%d): %s",
				finalRequest.Address,
				finalRequest.TransactionID,
				*failure,
			)
		default:
			// leave this case for ooni/pipeline
		}
		return
	}

	// fallback to the HTTP diff algo.
	tk.analysisHTTPDiff(logger, finalRequest, &ctrl)
}
