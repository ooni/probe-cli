package webconnectivitylte

//
// Core analysis
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// These flags determine the context of TestKeys.Blocking. However, while .Blocking
// is an enumeration, these flags allow to describe multiple blocking methods.
const (
	// AnalysisBlockingFlagDNSBlocking indicates there's blocking at the DNS level.
	AnalysisBlockingFlagDNSBlocking = 1 << iota

	// AnalysisBlockingFlagTCPIPBlocking indicates there's blocking at the TCP/IP level.
	AnalysisBlockingFlagTCPIPBlocking

	// AnalysisBlockingFlagTLSBlocking indicates there were TLS issues.
	AnalysisBlockingFlagTLSBlocking

	// AnalysisBlockingFlagHTTPBlocking indicates there was an HTTP failure.
	AnalysisBlockingFlagHTTPBlocking

	// AnalysisBlockingFlagHTTPDiff indicates there's an HTTP diff.
	AnalysisBlockingFlagHTTPDiff

	// AnalysisBlockingFlagSuccess indicates we did not detect any blocking.
	AnalysisBlockingFlagSuccess
)

// analysisToplevel is the toplevel function that analyses the results
// of the experiment once all network tasks have completed.
//
// This function sets v0.4-compatible test keys as well as v0.5-specific
// test keys that attempt to provide a more fine-grained view of the
// results, so that we can flag cases with multiple blocking scenarios.
//
// This function MUTATES the test keys.
func (tk *TestKeys) analysisToplevel(logger model.Logger) {
	analysisEngineClassic(tk, logger)
}

const (
	// AnalysisFlagNullNullExpectedDNSLookupFailure indicates some of the DNS lookup
	// attempts failed both in the probe and in the test helper.
	AnalysisFlagNullNullExpectedDNSLookupFailure = 1 << iota

	// AnalysisFlagNullNullExpectedTCPConnectFailure indicates that some of the connect
	// attempts failed both in the probe and in the test helper.
	AnalysisFlagNullNullExpectedTCPConnectFailure

	// AnalysisFlagNullNullExpectedTLSHandshakeFailure indicates that we have seen some TLS
	// handshakes failing consistently for both the probe and the TH.
	AnalysisFlagNullNullExpectedTLSHandshakeFailure

	// AnalysisFlagNullNullSuccessfulHTTPS indicates that we had no TH data
	// but all the HTTP requests used always HTTPS and never failed.
	AnalysisFlagNullNullSuccessfulHTTPS

	// AnalysisFlagNullNullUnexpectedDNSLookupSuccess indicates the case
	// where the TH resolved no addresses while the probe did.
	AnalysisFlagNullNullUnexpectedDNSLookupSuccess
)

const (
	// AnalysisFlagDNSBogon indicates we got any bogon reply
	AnalysisFlagDNSBogon = 1 << iota

	// AnalysisDNSFlagUnexpectedFailure indicates the TH could
	// resolve a domain while the probe couldn't
	AnalysisDNSFlagUnexpectedFailure

	// AnalysisDNSFlagUnexpectedAddrs indicates the TH resolved
	// different addresses from the probe
	AnalysisDNSFlagUnexpectedAddrs
)
