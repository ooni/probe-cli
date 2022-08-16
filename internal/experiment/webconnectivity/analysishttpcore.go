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
// - analysisFlagTLSBlocking
//
// - analysisFlagHTTPBlocking
//
// - analysisFlagHTTPDiff
//
// In websteps fashion, we don't stop at the first failure, rather we
// process all the available data and evaluate all possible errors.
func (tk *TestKeys) analysisHTTPToplevel(logger model.Logger) {
	// don't perform any analysis if the TH failed
	if tk.Control == nil {
		return
	}
	ctrl := tk.Control.HTTPRequest

	// don't perform any analysis if the TH's HTTP measurement failed
	if ctrl.Failure != nil {
		return
	}

	// determine whether we had any TLS handshake issue and, in such a case,
	// declare that we had a case of "http-failure" through TLS.
	//
	// Note that this would eventually count as an "http-failure" for .Blocking
	// because Web Connectivity did not have a concept of TLS based blocking.
	//
	// This check works ~reliably as long as we ensure to put DoH TLS
	// handshakes outside of the main .TLSHandshakes field.
	if tk.hasWellKnownTLSHandshakeIssues(logger) {
		tk.XBlockingFlags |= analysisFlagTLSBlocking
		// continue processing
	}

	// determine whether we had well known cleartext HTTP round trip issues
	// and, in such a case, declare we had an "http-failure".
	if tk.hasWellKnownHTTPRoundTripIssues(logger) {
		tk.XBlockingFlags |= analysisFlagHTTPBlocking
		// continue processing
	}

	// if we don't have any request to check, there's not much more we
	// can actually do here, so let's just return.
	if len(tk.Requests) <= 0 {
		return
	}

	// if the request has failed in any other way, we don't know. By convention, the first
	// entry in the tk.Requests array is the last entry that was measured.
	finalRequest := tk.Requests[0]
	if finalRequest.Failure != nil {
		return
	}

	// fallback to the HTTP diff algo.
	tk.analysisHTTPDiff(logger, finalRequest, &ctrl)
}

// hasWellKnownTLSHandshakeIssues returns true in case we observed
// a set of well-known issues during the TLS handshake.
func (tk *TestKeys) hasWellKnownTLSHandshakeIssues(logger model.Logger) (result bool) {
	// TODO(bassosimone): we should return TLS information in the TH
	// such that we can perform a TCP-like check
	for _, thx := range tk.TLSHandshakes {
		fail := thx.Failure
		if fail == nil {
			continue // this handshake succeded, so skip it
		}
		switch *fail {
		case netxlite.FailureConnectionReset,
			netxlite.FailureGenericTimeoutError,
			netxlite.FailureEOFError,
			netxlite.FailureSSLInvalidHostname,
			netxlite.FailureSSLInvalidCertificate,
			netxlite.FailureSSLUnknownAuthority:
			logger.Warnf(
				"TLS: endpoint %s fails with %s (see #%d)",
				thx.Address, *fail, thx.TransactionID,
			)
			result = true // flip the result but continue looping so we print them all
		default:
			// check next handshake
		}
	}
	return
}

// hasWellKnownHTTPRoundTripIssues checks whether any HTTP round
// trip failed in a well-known suspicious way
func (tk *TestKeys) hasWellKnownHTTPRoundTripIssues(logger model.Logger) (result bool) {
	for _, rtx := range tk.Requests {
		fail := rtx.Failure
		if fail == nil {
			// This one succeded, so skip it. Note that, in principle, we know
			// the fist entry is the last request occurred, but I really do not
			// want to embed this bad assumption in one extra place!
			continue
		}
		switch *fail {
		case netxlite.FailureConnectionReset,
			netxlite.FailureGenericTimeoutError,
			netxlite.FailureEOFError:
			logger.Warnf(
				"TLS: endpoint %s fails with %s (see #%d)",
				"N/A", *fail, rtx.TransactionID, // TODO(bassosimone): implement
			)
			result = true // flip the result but continue looping so we print them all
		default:
			// check next round trip
		}
	}
	return
}
