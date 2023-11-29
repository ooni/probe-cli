package webconnectivitylte

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func (tk *TestKeys) analysisToplevelV2(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// 1. produce observations using the minipipeline
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

	// 2. filter observations to only include results collected by the
	// system resolver, which approximates v0.4's results
	classic := minipipeline.ClassicFilter(container)

	// 3. run the classic analysis algorithm
	tk.analysisClassic(logger, classic)
}

func analysisValueToPointer[T any](input T) *T {
	return &input
}

func analysisOptionalToPointer[T any](input optional.Value[T]) *T {
	if input.IsNone() {
		return nil
	}
	return analysisValueToPointer(input.Unwrap())
}

func (tk *TestKeys) analysisClassic(logger model.Logger, container *minipipeline.WebObservationsContainer) {
	// dump the observations
	fmt.Printf("%s\n", must.MarshalJSON(container))

	// produce the woa based on the observations
	woa := minipipeline.AnalyzeWebObservations(container)

	// dump the analysis
	fmt.Printf("%s\n", must.MarshalJSON(woa))

	// determine the DNS consistency
	switch {
	case (woa.EndpointIPAddressesControlInvalid.Len() <= 0 &&
		woa.EndpointIPAddressesInvalidBogon.Len() <= 0 &&
		(woa.EndpointIPAddressesControlValidByASN.Len() > 0 ||
			woa.EndpointIPAddressesValidTLS.Len() > 0 ||
			woa.EndpointIPAddressesControlValidByEquality.Len() > 0)):
		// we have consistency if:
		//
		// 1. the control did not detect any invalid IP address; and
		//
		// 2. we did not see any bogon; and
		//
		// 3. we have at least one address valid by ASN, TLS, or with-control equality.
		tk.DNSConsistency = optional.Some("consistent")

	case (woa.EndpointIPAddressesControlInvalid.Len() > 0 ||
		woa.EndpointIPAddressesInvalidBogon.Len() > 0 ||
		woa.DNSLookupWithControlUnexpectedFailure.Len() > 0):
		// we have inconsistency if:
		//
		// 1. there is at least one address marked as invalid by the control; or
		//
		// 2. we did see at least a bogon; or
		//
		// 3. we have seen unexpected DNS failures.
		tk.DNSConsistency = optional.Some("inconsistent")

	default:
		tk.DNSConsistency = optional.None[string]()
	}

	// we must set blocking to "dns" when there's a DNS inconsistency
	setBlocking := func(value string) string {
		switch {
		case !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent":
			return "dns"
		default:
			return value
		}

	}

	// set HTTPDiff values
	if !woa.HTTPFinalResponseDiffBodyProportionFactor.IsNone() {
		tk.BodyLengthMatch = optional.Some(woa.HTTPFinalResponseDiffBodyProportionFactor.Unwrap() > 0.7)
	}
	if !woa.HTTPFinalResponseDiffUncommonHeadersIntersection.IsNone() {
		tk.HeadersMatch = optional.Some(woa.HTTPFinalResponseDiffUncommonHeadersIntersection.Unwrap().Len() > 0)
	}
	tk.StatusCodeMatch = woa.HTTPFinalResponseDiffStatusCodeMatch
	if !woa.HTTPFinalResponseDiffTitleDifferentLongWords.IsNone() {
		tk.TitleMatch = optional.Some(woa.HTTPFinalResponseDiffTitleDifferentLongWords.Unwrap().Len() <= 0)
	}

	// if we have a final HTTPS response, we're good
	if woa.HTTPFinalResponseWithoutControlTLS.Len() > 0 || woa.HTTPFinalResponseWithControlTLS.Len() > 0 {
		tk.Blocking = false
		tk.Accessible = true
		return
	}

	// if we have a final HTTP response with control, let's run HTTPDiff
	if woa.HTTPFinalResponseWithControlTCP.Len() > 0 {
		if !tk.StatusCodeMatch.IsNone() && tk.StatusCodeMatch.Unwrap() {
			if !tk.BodyLengthMatch.IsNone() && tk.BodyLengthMatch.Unwrap() {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
			if !tk.HeadersMatch.IsNone() && tk.HeadersMatch.Unwrap() {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
			if !tk.TitleMatch.IsNone() && tk.TitleMatch.Unwrap() {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
			// fallthrough
		}
		tk.Blocking = setBlocking("http-diff")
		tk.Accessible = false
		return
	}

	// if we have a final HTTP response without control, we don't know
	if woa.HTTPFinalResponseWithoutControlTCP.Len() > 0 {
		return
	}

	// if we have unexpected HTTP round trip failures, it's "http-failure"
	if woa.HTTPNonFinalResponseFailureWithControlUnexpected.Len() > 0 {
		tk.Blocking = setBlocking("http-failure")
		tk.Accessible = false
		return
	}

	// if we have unexpected TCP connect failures, it's "tcp_ip"
	//
	// we give precendence to this over TLS handshake because the latter
	// always produces "http-failure", which we handle below
	if woa.TCPConnectWithControlUnexpectedFailureAnomaly.Len() > 0 {
		tk.Blocking = setBlocking("tcp_ip")
		tk.Accessible = false
		return
	}

	// if we have unexpected TLS failures, it's "http-failure"
	if woa.TLSHandshakeWithControlUnexpectedFailure.Len() > 0 {
		tk.Blocking = setBlocking("http-failure")
		tk.Accessible = false
		return
	}

	// if we have unexplained TCP failures, blame "http-failure"
	if woa.TCPConnectWithoutControlFailure.Len() > 0 {
		tk.Blocking = setBlocking("http-failure")
		tk.Accessible = false
		return
	}

	// likewise but for unexplained TLS handshake failures
	if woa.TLSHandshakeWithoutControlFailure.Len() > 0 {
		tk.Blocking = setBlocking("http-failure")
		tk.Accessible = false
		return
	}

	// if we arrive here and the DNS is still inconsistent, say "dns"
	if !tk.DNSConsistency.IsNone() && tk.DNSConsistency.Unwrap() == "inconsistent" {
		tk.Blocking = "dns"
		tk.Accessible = false
		return
	}

	// otherwise, we don't know what to say
}

/*

// analysisToplevelV2 is an alternative version of the analysis code that
// uses the [minipipeline] package for processing.
func (tk *TestKeys) analysisToplevelV2Old(logger model.Logger) {
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
	analysis.ComputeHTTPFinalResponsesWithControl(container)
	analysis.ComputeTCPTransactionsWithUnexplainedUnexplainedFailures(container)
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
		v := "inconsistent"
		tk.DNSConsistency = &v
		tk.BlockingFlags |= analysisFlagDNSBlocking
	} else {
		logger.Info("DNSConsistency: consistent")
		v := "consistent"
		tk.DNSConsistency = &v
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
	numFinals := len(analysis.HTTPFinalResponsesWithControl.UnwrapOr(nil))
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
*/
