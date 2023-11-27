package webconnectivitylte

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// analysisToplevelV2 is an alternative version of the analysis code that
// uses the [minipipeline] package for processing.
func (tk *TestKeys) analysisToplevelV2(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	container := minipipeline.NewWebObservationsContainer()
	container.IngestDNSLookupEvents(tk.Queries...)
	container.IngestTCPConnectEvents(tk.TCPConnect...)
	container.IngestTLSHandshakeEvents(tk.TLSHandshakes...)
	container.IngestHTTPRoundTripEvents(tk.Requests...)

	// be defensive in case the control request or control are not defined
	if tk.ControlRequest != nil && tk.Control != nil {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have triggered previous errors
		runtimex.Try0(container.IngestControlMessages(tk.ControlRequest, tk.Control))
	}

	// dump the pipeline results for debugging purposes
	fmt.Printf("%s\n", must.MarshalJSON(container))

	// produce the analysis based on the observations
	analysis := &minipipeline.WebAnalysis{}
	analysis.ComputeDNSExperimentFailure(container)
	analysis.ComputeDNSTransactionsWithBogons(container)
	analysis.ComputeDNSTransactionsWithUnexpectedFailures(container)
	analysis.ComputeDNSPossiblyInvalidAddrs(container)
	analysis.ComputeTCPTransactionsWithUnexpectedTCPConnectFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedHTTPFailures(container)
	analysis.ComputeHTTPDiffBodyProportionFactor(container)
	analysis.ComputeHTTPDiffStatusCodeMatch(container)
	analysis.ComputeHTTPDiffUncommonHeadersIntersection(container)
	analysis.ComputeHTTPDiffTitleDifferentLongWords(container)
	analysis.ComputeHTTPFinalResponses(container)
	analysis.ComputeTCPTransactionsWithUnexplainedUnexpectedFailures(container)
	analysis.ComputeHTTPFinalResponsesWithTLS(container)

	// dump the analysis results for debugging purposes
	fmt.Printf("%s\n", must.MarshalJSON(analysis))

	// run top-level protocol-specific analysis algorithms
	tk.analysisDNSToplevelV2(analysis, logger)
	tk.analysisTCPIPToplevelV2(analysis, logger)
	tk.analysisTLSToplevelV2(analysis, logger)
	tk.analysisHTTPToplevelV2(analysis, logger)
}

// analysisDNSToplevelV2 is the toplevel analysis function for DNS results.
//
// Note: this function DOES NOT consider failed DNS-over-HTTPS (DoH) submeasurements
// and ONLY considers the IP addrs they have resolved. Failing to contact a DoH service
// provides info about such a DoH service rather than on the measured URL. See the
// https://github.com/ooni/probe/issues/2274 issue for more info.
//
// The goals of this function are the following:
//
// 1. Set the legacy .DNSExperimentFailure field to the failure value of the
// first DNS query that failed among the ones using getaddrinfo. This field is
// legacy because now we perform several DNS lookups.
//
// 2. Compute the XDNSFlags value.
//
// From the XDNSFlags value, we determine, in turn DNSConsistency and
// XBlockingFlags according to the following decision table:
//
//	+-----------+----------------+---------------------+
//	| XDNSFlags | DNSConsistency | XBlockingFlags      |
//	+-----------+----------------+---------------------+
//	| 0         | "consistent"   | no change           |
//	+-----------+----------------+---------------------+
//	| nonzero   | "inconsistent" | set FlagDNSBlocking |
//	+-----------+----------------+---------------------+
//
// We explain how XDNSFlags is determined in the documentation of
// the functions that this function calls to do its job.
func (tk *TestKeys) analysisDNSToplevelV2(analysis *minipipeline.WebAnalysis, logger model.Logger) {
	// TODO(bassosimone): we should probably keep logging as before here

	// assign flags depending on the analysis
	if v := analysis.DNSExperimentFailure.UnwrapOr(""); v != "" {
		tk.DNSExperimentFailure = &v
	}
	if v := analysis.DNSTransactionsWithBogons.UnwrapOr(nil); len(v) > 0 {
		tk.DNSFlags |= AnalysisDNSBogon
	}
	if v := analysis.DNSTransactionsWithUnexpectedFailures.UnwrapOr(nil); len(v) > 0 {
		tk.DNSFlags |= AnalysisDNSUnexpectedFailure
	}
	if v := analysis.DNSPossiblyInvalidAddrs.UnwrapOr(nil); len(v) > 0 {
		tk.DNSFlags |= AnalysisDNSUnexpectedAddrs
	}

	// compute DNS consistency
	if tk.DNSFlags != 0 {
		logger.Warn("DNSConsistency: inconsistent")
		tk.DNSConsistency = "inconsistent"
		tk.BlockingFlags |= analysisFlagDNSBlocking
	} else {
		logger.Info("DNSConsistency: consistent")
		tk.DNSConsistency = "consistent"
	}
}

// analysisTCPIPToplevelV2 is the toplevel analysis function for TCP/IP results.
func (tk *TestKeys) analysisTCPIPToplevelV2(analysis *minipipeline.WebAnalysis, logger model.Logger) {
	// TODO(bassosimone): we should probably keep logging as before here
	// TODO(bassosimone): we're ignoring .Status here

	if v := analysis.TCPTransactionsWithUnexpectedTCPConnectFailures.UnwrapOr(nil); len(v) > 0 {
		tk.BlockingFlags |= analysisFlagTCPIPBlocking
	}
}

// analysisTLSToplevelV2 is the toplevel analysis function for TLS results.
func (tk *TestKeys) analysisTLSToplevelV2(analysis *minipipeline.WebAnalysis, logger model.Logger) {
	// TODO(bassosimone): we should probably keep logging as before here

	if v := analysis.TCPTransactionsWithUnexpectedTLSHandshakeFailures.UnwrapOr(nil); len(v) > 0 {
		tk.BlockingFlags |= analysisFlagTLSBlocking
	}
}

// analysisHTTPToplevelV2 is the toplevel analysis function for HTTP results.
func (tk *TestKeys) analysisHTTPToplevelV2(analysis *minipipeline.WebAnalysis, logger model.Logger) {
	// TODO(bassosimone): we should probably keep logging as before here
	// TODO(bassosimone): we're missing the success with HTTPS flag...

	if v := analysis.TCPTransactionsWithUnexpectedHTTPFailures.UnwrapOr(nil); len(v) > 0 {
		tk.BlockingFlags |= analysisFlagHTTPBlocking
	}

	// Detect cases where an error occurred during a redirect. For this to happen, we
	// need to observe (1) no "final" responses and (2) unexpected, unexplained failures
	numFinals := len(analysis.HTTPFinalResponses.UnwrapOr(nil))
	numUnexpectedUnexplained := len(analysis.TCPTransactionsWithUnexplainedUnexpectedFailures.UnwrapOr(nil))
	if numFinals <= 0 && numUnexpectedUnexplained > 0 {
		tk.BlockingFlags |= analysisFlagHTTPBlocking
	}

	// Special case for HTTPS
	if len(analysis.HTTPFinalResponsesWithTLS.UnwrapOr(nil)) > 0 {
		tk.BlockingFlags |= analysisFlagSuccess
	}

	// attempt to fill the comparisons about the body
	//
	// XXX this code should probably always run
	if !analysis.HTTPDiffStatusCodeMatch.IsNone() {
		value := analysis.HTTPDiffStatusCodeMatch.Unwrap()
		tk.StatusCodeMatch = &value
	}
	if !analysis.HTTPDiffBodyProportionFactor.IsNone() {
		value := analysis.HTTPDiffBodyProportionFactor.UnwrapOr(0) > 0.7
		tk.BodyLengthMatch = &value
	}
	if !analysis.HTTPDiffUncommonHeadersIntersection.IsNone() {
		value := len(analysis.HTTPDiffUncommonHeadersIntersection.Unwrap()) > 0
		tk.HeadersMatch = &value
	}
	if !analysis.HTTPDiffTitleDifferentLongWords.IsNone() {
		value := len(analysis.HTTPDiffTitleDifferentLongWords.Unwrap()) <= 0
		tk.TitleMatch = &value
	}

	// same code structure as before
	if !analysis.HTTPDiffStatusCodeMatch.IsNone() {
		if analysis.HTTPDiffStatusCodeMatch.Unwrap() {

			if analysis.HTTPDiffBodyProportionFactor.UnwrapOr(0) > 0.7 {
				tk.BlockingFlags |= analysisFlagSuccess
				return
			}

			if v := analysis.HTTPDiffUncommonHeadersIntersection.UnwrapOr(nil); len(v) > 0 {
				tk.BlockingFlags |= analysisFlagSuccess
				return
			}

			if !analysis.HTTPDiffTitleDifferentLongWords.IsNone() &&
				len(analysis.HTTPDiffTitleDifferentLongWords.Unwrap()) <= 0 {
				tk.BlockingFlags |= analysisFlagSuccess
				return
			}
		}
		tk.BlockingFlags |= analysisFlagHTTPDiff
	}
}
