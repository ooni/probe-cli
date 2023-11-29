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

	// produce the analysis based on the observations
	analysis := minipipeline.AnalyzeWebObservations(container)

	// dump the analysis
	fmt.Printf("%s\n", must.MarshalJSON(analysis))

	// set DNSExperimentFailure
	tk.DNSExperimentFailure = analysisOptionalToPointer(analysis.DNSExperimentFailure)

	// set DNSConsistency
	var (
		dnsTransactionsWithUnexpectedFailures *bool
		dnsPossiblyInvalidAddrs               *bool
		dnsBogonsFailure                      *bool
	)
	if v := analysis.DNSTransactionsWithUnexpectedFailures.UnwrapOr(nil); v != nil {
		g := len(v) > 0
		dnsTransactionsWithUnexpectedFailures = &g
	}
	if v := analysis.DNSPossiblyInvalidAddrsClassic.UnwrapOr(nil); v != nil {
		g := len(v) > 0
		dnsPossiblyInvalidAddrs = &g
	}
	if v := analysis.DNSTransactionsWithBogons.UnwrapOr(nil); v != nil {
		g := len(v) > 0
		dnsBogonsFailure = &g
	}
	tk.DNSConsistency = func() *string {
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			v := "inconsistent"
			return &v
		}
		if dnsPossiblyInvalidAddrs == nil && dnsTransactionsWithUnexpectedFailures == nil {
			// this models the case where the control fails, which is when v0.4 does not set
			// anything for dnsconsistency, and we should do the same
			//
			// the check for bogons is actually useful because the way in which we test
			// for asn compatibility here is different and does not consider the case
			// where the bogon returns the zero ASN
			//
			// in such a case though only dnsTransactionsWithUnexpectedFailures would be nil
			// and we would go ahead and also check for bogon
			//
			// BUT NO!!!! AND SO WE NEED THE BOGON IN FRONT OF THIS SHIT
			return nil
		}
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			v := "inconsistent"
			return &v
		}
		if dnsTransactionsWithUnexpectedFailures != nil && *dnsTransactionsWithUnexpectedFailures {
			v := "inconsistent"
			return &v
		}
		v := "consistent"
		return &v
	}()

	// FIXME: HTTPExperimentFailure

	// set BodyLengthMatch
	if v := analysis.HTTPDiffBodyProportionFactor.UnwrapOr(0); v > 0 {
		tk.BodyLengthMatch = analysisValueToPointer(v > 0.7)
	}

	// set HeadersMatch
	if v := analysis.HTTPDiffUncommonHeadersIntersection.UnwrapOr(nil); v != nil {
		tk.HeadersMatch = analysisValueToPointer(len(v) > 0)
	}

	// set StatusCodeMatch
	tk.StatusCodeMatch = analysisOptionalToPointer(analysis.HTTPDiffStatusCodeMatch)

	// set TitleMatch
	if v := analysis.HTTPDiffTitleDifferentLongWords.UnwrapOr(nil); v != nil {
		tk.TitleMatch = analysisValueToPointer(len(v) <= 0)
	}

	//
	// set Blocking & Accessible
	//

	// if we have a final response using TLS, we declare there's no blocking
	if v := analysis.HTTPFinalResponsesWithTLS.UnwrapOr(nil); len(v) > 0 {
		tk.Blocking = false
		tk.Accessible = true
		return
	}

	// otherwise, if we have a final response, let's execute the dns-diff algorithm
	if v := analysis.HTTPFinalResponsesWithControl.UnwrapOr(nil); len(v) > 0 {
		if tk.StatusCodeMatch != nil && *tk.StatusCodeMatch {
			if tk.BodyLengthMatch != nil && *tk.BodyLengthMatch {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
			if tk.HeadersMatch != nil && *tk.HeadersMatch {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
			if tk.TitleMatch != nil && *tk.TitleMatch {
				tk.Blocking = false
				tk.Accessible = true
				return
			}
		}
		// It seems we didn't get the expected web page. What now? Well, if
		// the DNS does not seem trustworthy, let us blame it.
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		// The only remaining conclusion seems that the web page we have got
		// doesn't match what we were expecting.
		tk.Blocking = "http-diff"
		tk.Accessible = false
		return
	}

	// otherwise we need to determine whether it's http-failure or tcp_ip
	// with dns always being a possibility to consider
	//
	// we start by checking for unexpected TLS failures, which we map as
	// http-failure since there's no TLS failure for v0.4
	//
	// note that this kind of failures happen in the first request, so
	// we can be confident and apply a DNS correction here
	if v := analysis.TCPTransactionsWithUnexpectedTLSHandshakeFailures.UnwrapOr(nil); len(v) > 0 {
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		tk.Blocking = "http-failure"
		tk.Accessible = false
		return
	}

	if v := analysis.TCPTransactionsWithUnexpectedTCPConnectFailures.UnwrapOr(nil); len(v) > 0 {
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		tk.Blocking = "tcp_ip"
		tk.Accessible = false
		return
	}

	// then we check for unexpected HTTP failures in the first request
	if v := analysis.TCPTransactionsWithUnexpectedHTTPFailures.UnwrapOr(nil); len(v) > 0 {
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		tk.Blocking = "http-failure"
		tk.Accessible = false
		return
	}

	// then we check for unexpected HTTP failures in the first request
	if v := analysis.TCPTransactionsWithUnexplainedUnexpectedFailures.UnwrapOr(nil); len(v) > 0 {
		if dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		if dnsBogonsFailure != nil && *dnsBogonsFailure {
			tk.Blocking = "dns"
			tk.Accessible = false
			return
		}
		tk.Blocking = "http-failure"
		tk.Accessible = false
		return
	}

	// fallback to DNS if we don't know exactly what to do
	if (dnsPossiblyInvalidAddrs != nil && *dnsPossiblyInvalidAddrs) ||
		(dnsTransactionsWithUnexpectedFailures != nil && *dnsTransactionsWithUnexpectedFailures) {
		tk.Blocking = "dns"
		tk.Accessible = false
	}

	// what remains here are all the cases when the website is down
	//
	// the first case we consider is NXDOMAIN
	if v := analysis.DNSPossiblyNonexistingDomains.UnwrapOr(nil); len(v) > 0 {
		// TODO(bassosimone): this is a condition where v0.4 is actually wrong but
		// we should keep its wrong behavior for now. The correct result here would
		// actually be to set Accessible to false.
		tk.Blocking = false
		tk.Accessible = true
	}
}

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
