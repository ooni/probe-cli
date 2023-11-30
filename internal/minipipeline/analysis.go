package minipipeline

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// AnalyzeWebObservations generates a [*WebAnalysis] from a [*WebObservationsContainer].
func AnalyzeWebObservations(container *WebObservationsContainer) *WebAnalysis {
	analysis := &WebAnalysis{}

	analysis.dnsComputeSuccessMetrics(container)
	analysis.dnsComputeSuccessMetricsClassic(container)
	analysis.dnsComputeFailureMetrics(container)

	analysis.tcpComputeMetrics(container)
	analysis.tlsComputeMetrics(container)
	analysis.httpComputeFailureMetrics(container)
	analysis.httpComputeFinalResponseMetrics(container)

	analysis.ComputeDNSExperimentFailure(container)
	analysis.ComputeDNSPossiblyNonexistingDomains(container)

	return analysis
}

// WebAnalysis summarizes the content of [*WebObservationsContainer].
//
// The zero value of this struct is ready to use.
type WebAnalysis struct {
	// DNSLookupSuccessWithInvalidAddresses contains DNS transactions with invalid IP addresses by
	// taking into account control info, bogons, and TLS handshakes.
	DNSLookupSuccessWithInvalidAddresses Set[int64]

	// DNSLookupSuccessWithInvalidAddressesClassic is like DNSLookupInvalid but the algorithm is more relaxed
	// to be compatible with Web Connectivity v0.4's behavior.
	DNSLookupSuccessWithInvalidAddressesClassic Set[int64]

	// DNSLookupUnexpectedFailure contains DNS transactions with unexpected failures.
	DNSLookupUnexpectedFailure Set[int64]

	// TCPConnectUnexpectedFailure contains TCP endpoint transactions with unexpected failures.
	TCPConnectUnexpectedFailure Set[int64]

	// TCPConnectUnexpectedFailureDuringWebFetch contains TCP endpoint transactions with unexpected failures
	// while performing a web fetch, as opposed to checking for connectivity.
	TCPConnectUnexpectedFailureDuringWebFetch Set[int64]

	// TCPConnectUnexpectedFailureDuringConnectivityCheck contains TCP endpoint transactions with unexpected failures
	// while checking for connectivity, as opposed to fetching a webpage.
	TCPConnectUnexpectedFailureDuringConnectivityCheck Set[int64]

	// TCPConnectUnexplainedFailure contains failures occurring during redirects.
	TCPConnectUnexplainedFailure Set[int64]

	// TCPConnectUnexplainedFailureDuringWebFetch contains failures occurring during redirects
	// while performing a web fetch, as opposed to checking for connectivity.
	TCPConnectUnexplainedFailureDuringWebFetch Set[int64]

	// TCPConnectUnexplainedFailureDuringConnectivityCheck contains failures occurring during redirects
	// while checking for connectivity, as opposed to fetching a webpage.
	TCPConnectUnexplainedFailureDuringConnectivityCheck Set[int64]

	// TLSHandshakeUnexpectedFailure contains TLS endpoint transactions with unexpected failures.
	TLSHandshakeUnexpectedFailure Set[int64]

	// TLSHandshakeUnexpectedFailureDuringWebFetch contains TLS endpoint transactions with unexpected failures.
	// while performing a web fetch, as opposed to checking for connectivity.
	TLSHandshakeUnexpectedFailureDuringWebFetch Set[int64]

	// TLSHandshakeUnexpectedFailureDuringConnectivityCheck contains TLS endpoint transactions with unexpected failures.
	// while checking for connectivity, as opposed to fetching a webpage.
	TLSHandshakeUnexpectedFailureDuringConnectivityCheck Set[int64]

	// TLSHandshakeUnexplainedFailure contains failures occurring during redirects.
	TLSHandshakeUnexplainedFailure Set[int64]

	// TLSHandshakeUnexplainedFailureDuringWebFetch  contains failures occurring during redirects
	// while performing a web fetch, as opposed to checking for connectivity.
	TLSHandshakeUnexplainedFailureDuringWebFetch Set[int64]

	// TLSHandshakeUnexplainedFailureDuringConnectivityCheck contains failures occurring during redirects
	// while checking for connectivity, as opposed to fetching a webpage.
	TLSHandshakeUnexplainedFailureDuringConnectivityCheck Set[int64]

	// HTTPRoundTripUnexpectedFailure contains HTTP endpoint transactions with unexpected failures.
	HTTPRoundTripUnexpectedFailure Set[int64]

	// HTTPFinalResponseSuccessTLSWithoutControl contains the ID of the final response
	// transaction when the final response succeeded without control and with TLS.
	HTTPFinalResponseSuccessTLSWithoutControl optional.Value[int64]

	// HTTPFinalResponseSuccessTLSWithControl contains the ID of the final response
	// transaction when the final response succeeded with control and with TLS.
	HTTPFinalResponseSuccessTLSWithControl optional.Value[int64]

	// HTTPFinalResponseSuccessTCPWithControl contains the ID of the final response
	// transaction when the final response succeeded with control and with TCP.
	HTTPFinalResponseSuccessTCPWithControl optional.Value[int64]

	// HTTPDiffBodyProportionFactor is the body proportion factor.
	HTTPDiffBodyProportionFactor optional.Value[float64]

	// HTTPDiffStatusCodeMatch returns whether the status code matches.
	HTTPDiffStatusCodeMatch optional.Value[bool]

	// HTTPDiffTitleDifferentLongWords contains the words long 5+ characters that appear
	// in the probe's "final" response title or in the TH title but not in both.
	HTTPDiffTitleDifferentLongWords optional.Value[map[string]bool]

	// HTTPDiffUncommonHeadersIntersection contains the uncommon headers intersection.
	HTTPDiffUncommonHeadersIntersection optional.Value[map[string]bool]

	// TODO(bassosimone): there are probably redundant metrics from this point on

	// DNSExperimentFailure is the first failure experienced by a getaddrinfo-like resolver.
	DNSExperimentFailure optional.Value[string]

	// DNSPossiblyNonexistingDomains lists all the domains for which both
	// the probe and the TH failed to perform DNS lookups.
	DNSPossiblyNonexistingDomains optional.Value[map[string]bool]
}

func (wa *WebAnalysis) dnsComputeSuccessMetrics(c *WebObservationsContainer) {
	// fill the invalid set
	var already Set[int64]
	for _, obs := range c.DNSLookupSuccesses {
		// avoid considering a lookup we already considered
		if already.Contains(obs.DNSTransactionID.Unwrap()) {
			continue
		}
		already.Add(obs.DNSTransactionID.Unwrap())

		// lookups once we started following redirects should not be considered
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			continue
		}

		// if there's a bogon, mark as invalid
		if !obs.IPAddressBogon.IsNone() && obs.IPAddressBogon.Unwrap() {
			wa.DNSLookupSuccessWithInvalidAddresses.Add(obs.DNSTransactionID.Unwrap())
			continue
		}

		// when there is no control info, we cannot say much
		if obs.ControlDNSResolvedAddrs.IsNone() {
			continue
		}

		// obtain measurement and control
		measurement := obs.DNSResolvedAddrs.Unwrap()
		control := obs.ControlDNSResolvedAddrs.Unwrap()

		// this lookup is good if there is IP addresses intersection
		if DNSDiffFindCommonIPAddressIntersection(measurement, control).Len() > 0 {
			continue
		}

		// this lookup is good if there is ASN intersection
		if DNSDiffFindCommonASNsIntersection(measurement, control).Len() > 0 {
			continue
		}

		// mark as invalid
		wa.DNSLookupSuccessWithInvalidAddresses.Add(obs.DNSTransactionID.Unwrap())
	}

	// undo using TLS handshake info
	for _, obs := range c.KnownTCPEndpoints {
		// we must have a successuful TLS handshake
		if obs.TLSHandshakeFailure.IsNone() || obs.TLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		// we must have a DNSTransactionID
		txid := obs.DNSTransactionID.UnwrapOr(0)
		if txid <= 0 {
			continue
		}

		// this is actually valid
		wa.DNSLookupSuccessWithInvalidAddresses.Remove(txid)
	}
}

func (wa *WebAnalysis) dnsComputeSuccessMetricsClassic(c *WebObservationsContainer) {
	var already Set[int64]

	for _, obs := range c.DNSLookupSuccesses {
		// avoid considering a lookup we already considered
		if already.Contains(obs.DNSTransactionID.Unwrap()) {
			continue
		}
		already.Add(obs.DNSTransactionID.Unwrap())

		// lookups once we started following redirects should not be considered
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			continue
		}

		// when there is no control info, we cannot say much
		if obs.ControlDNSResolvedAddrs.IsNone() {
			continue
		}

		// obtain measurement and control
		measurement := obs.DNSResolvedAddrs.Unwrap()
		control := obs.ControlDNSResolvedAddrs.Unwrap()

		// this lookup is good if there is IP addresses intersection
		if DNSDiffFindCommonIPAddressIntersection(measurement, control).Len() > 0 {
			continue
		}

		// this lookup is good if there is ASN intersection
		if DNSDiffFindCommonASNsIntersection(measurement, control).Len() > 0 {
			continue
		}

		// mark as invalid
		wa.DNSLookupSuccessWithInvalidAddressesClassic.Add(obs.DNSTransactionID.Unwrap())
	}
}

func (wa *WebAnalysis) dnsComputeFailureMetrics(c *WebObservationsContainer) {
	var already Set[int64]

	for _, obs := range c.DNSLookupFailures {
		// avoid considering a lookup we already considered
		if already.Contains(obs.DNSTransactionID.Unwrap()) {
			continue
		}
		already.Add(obs.DNSTransactionID.Unwrap())

		// lookups once we started following redirects should not be considered
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			continue
		}

		// Implementation note: a DoH failure is not information about the URL we're
		// measuring but about the DoH service being blocked.
		//
		// See https://github.com/ooni/probe/issues/2274
		if utilsDNSEngineIsDNSOverHTTPS(obs) {
			continue
		}

		// skip cases where there's no DNS record for AAAA, which is a false positive
		if utilsDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
			continue
		}

		// TODO(bassosimone): if we set an IPv6 address as the resolver address, we
		// end up with false positive errors when there's no IPv6 support

		// handle the case where there's no control
		if obs.ControlDNSLookupFailure.IsNone() {
			continue
		}

		// handle the case where both failed
		if obs.DNSLookupFailure.Unwrap() != "" && obs.ControlDNSLookupFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the control failed
		if obs.ControlDNSLookupFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the probe failed
		if obs.DNSLookupFailure.Unwrap() != "" {
			wa.DNSLookupUnexpectedFailure.Add(obs.DNSTransactionID.Unwrap())
			continue
		}
	}
}

func (wa *WebAnalysis) tcpComputeMetrics(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// handle the case where there is no measurement
		if obs.TCPConnectFailure.IsNone() {
			continue
		}

		// dials once we started following redirects should be treated differently
		// since we know there's no control information beyond depth==0
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			if utilsTCPConnectFailureSeemsMisconfiguredIPv6(obs) {
				continue
			}
			if obs.TCPConnectFailure.Unwrap() != "" {
				switch {
				case !obs.TagFetchBody.IsNone() && obs.TagFetchBody.Unwrap():
					wa.TCPConnectUnexplainedFailureDuringWebFetch.Add(obs.EndpointTransactionID.Unwrap())
				case !obs.TagFetchBody.IsNone() && !obs.TagFetchBody.Unwrap():
					wa.TCPConnectUnexplainedFailureDuringConnectivityCheck.Add(obs.EndpointTransactionID.Unwrap())
				}
				wa.TCPConnectUnexplainedFailure.Add(obs.EndpointTransactionID.Unwrap())
				continue
			}
			continue
		}

		// handle the case where there is no control information
		if obs.ControlTCPConnectFailure.IsNone() {
			continue
		}

		// handle the case where both the probe and the control fail
		if obs.TCPConnectFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the control fails
		if obs.ControlTCPConnectFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the probe fails
		if obs.TCPConnectFailure.Unwrap() != "" {
			if utilsTCPConnectFailureSeemsMisconfiguredIPv6(obs) {
				continue
			}
			switch {
			case !obs.TagFetchBody.IsNone() && obs.TagFetchBody.Unwrap():
				wa.TCPConnectUnexpectedFailureDuringWebFetch.Add(obs.EndpointTransactionID.Unwrap())
			case !obs.TagFetchBody.IsNone() && !obs.TagFetchBody.Unwrap():
				wa.TCPConnectUnexpectedFailureDuringConnectivityCheck.Add(obs.EndpointTransactionID.Unwrap())
			}
			wa.TCPConnectUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
			continue
		}
	}
}

func (wa *WebAnalysis) tlsComputeMetrics(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// handle the case where there is no measurement
		if obs.TLSHandshakeFailure.IsNone() {
			continue
		}

		// handshakes once we started following redirects should be treated differently
		// since we know there's no control information beyond depth==0
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			if obs.TLSHandshakeFailure.Unwrap() != "" {
				switch {
				case !obs.TagFetchBody.IsNone() && obs.TagFetchBody.Unwrap():
					wa.TLSHandshakeUnexplainedFailureDuringWebFetch.Add(obs.EndpointTransactionID.Unwrap())
				case !obs.TagFetchBody.IsNone() && !obs.TagFetchBody.Unwrap():
					wa.TLSHandshakeUnexplainedFailureDuringConnectivityCheck.Add(obs.EndpointTransactionID.Unwrap())
				}
				wa.TLSHandshakeUnexplainedFailure.Add(obs.EndpointTransactionID.Unwrap())
				continue
			}
			continue
		}

		// handle the case where there is no control information
		if obs.ControlTLSHandshakeFailure.IsNone() {
			continue
		}

		// handle the case where both the probe and the control fail
		if obs.TLSHandshakeFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the control fails
		if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the probe fails
		if obs.TLSHandshakeFailure.Unwrap() != "" {
			switch {
			case !obs.TagFetchBody.IsNone() && obs.TagFetchBody.Unwrap():
				wa.TLSHandshakeUnexpectedFailureDuringWebFetch.Add(obs.EndpointTransactionID.Unwrap())
			case !obs.TagFetchBody.IsNone() && !obs.TagFetchBody.Unwrap():
				wa.TLSHandshakeUnexpectedFailureDuringConnectivityCheck.Add(obs.EndpointTransactionID.Unwrap())
			}
			wa.TLSHandshakeUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
			continue
		}
	}
}

func (wa *WebAnalysis) httpComputeFailureMetrics(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// Implementation note: here we don't limit the search to depth==0 because the
		// control we have for HTTP is relative to the final response.

		// handle the case where there is no measurement
		if obs.HTTPFailure.IsNone() {
			continue
		}

		// handle the case where there is no control information
		if obs.ControlHTTPFailure.IsNone() {
			continue
		}

		// handle the case where both the probe and the control fail
		if obs.HTTPFailure.Unwrap() != "" && obs.ControlHTTPFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the control fails
		if obs.ControlHTTPFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the probe fails
		if obs.HTTPFailure.Unwrap() != "" {
			wa.HTTPRoundTripUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
			continue
		}
	}
}

func (wa *WebAnalysis) httpComputeFinalResponseMetrics(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// we need a final HTTP response
		if obs.HTTPResponseIsFinal.IsNone() || !obs.HTTPResponseIsFinal.Unwrap() {
			continue
		}

		// stop after processing the first final response (there's at most
		// one when we're analyzing LTE results)
		wa.httpHandleFinalResponse(obs)
		return
	}
}

func (wa *WebAnalysis) httpHandleFinalResponse(obs *WebObservation) {
	// handle the case where there's no control
	if obs.ControlHTTPFailure.IsNone() {
		if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
			wa.HTTPFinalResponseSuccessTLSWithoutControl = obs.EndpointTransactionID
			return
		}
		return
	}

	// count and classify the number of final responses with control
	if !obs.TLSHandshakeFailure.IsNone() {
		runtimex.Assert(obs.TLSHandshakeFailure.Unwrap() == "", "expected to see TLS handshake success here")
		wa.HTTPFinalResponseSuccessTLSWithControl = obs.EndpointTransactionID
	} else {
		wa.HTTPFinalResponseSuccessTCPWithControl = obs.EndpointTransactionID
	}

	// compute the HTTPDiff metrics
	wa.httpDiffBodyProportionFactor(obs)
	wa.httpDiffStatusCodeMatch(obs)
	wa.httpDiffUncommonHeadersIntersection(obs)
	wa.httpDiffTitleDifferentLongWords(obs)
}

func (wa *WebAnalysis) httpDiffBodyProportionFactor(obs *WebObservation) {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return
	}

	// we need a valid body length and the body must not be truncated
	measurement := obs.HTTPResponseBodyLength.UnwrapOr(0)
	if measurement <= 0 || obs.HTTPResponseBodyIsTruncated.UnwrapOr(true) {
		return
	}

	// we also need a valid control body length
	control := obs.ControlHTTPResponseBodyLength.UnwrapOr(0)
	if control <= 0 {
		return
	}

	// compute the body proportion factor and update the state
	proportion := ComputeHTTPDiffBodyProportionFactor(measurement, control)
	wa.HTTPDiffBodyProportionFactor = optional.Some(proportion)
}

func (wa *WebAnalysis) httpDiffStatusCodeMatch(obs *WebObservation) {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return
	}

	// we need a positive status code for both
	measurement := obs.HTTPResponseStatusCode.UnwrapOr(0)
	if measurement <= 0 {
		return
	}
	control := obs.ControlHTTPResponseStatusCode.UnwrapOr(0)
	if control <= 0 {
		return
	}

	// update state
	wa.HTTPDiffStatusCodeMatch = ComputeHTTPDiffStatusCodeMatch(measurement, control)
}

func (wa *WebAnalysis) httpDiffUncommonHeadersIntersection(obs *WebObservation) {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return
	}

	// We should only perform the comparison if we have valid control data. Because
	// the headers could legitimately be empty, let's use the status code here.
	if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
		return
	}

	// Implementation note: here we need to continue running when either
	// headers are empty in order to produce an empty intersection. If we'd stop
	// after noticing that either dictionary is empty, we'd produce a nil
	// analysis result, which causes QA differences with v0.4.
	measurement := obs.HTTPResponseHeadersKeys.UnwrapOr(nil)
	control := obs.ControlHTTPResponseHeadersKeys.UnwrapOr(nil)

	state := ComputeHTTPDiffUncommonHeadersIntersection(measurement, control)
	wa.HTTPDiffUncommonHeadersIntersection = optional.Some(state)
}

func (wa *WebAnalysis) httpDiffTitleDifferentLongWords(obs *WebObservation) {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return
	}

	// We should only perform the comparison if we have valid control data. Because
	// the title could legitimately be empty, let's use the status code here.
	if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
		return
	}

	measurement := obs.HTTPResponseTitle.UnwrapOr("")
	control := obs.ControlHTTPResponseTitle.UnwrapOr("")

	state := ComputeHTTPDiffTitleDifferentLongWords(measurement, control)

	wa.HTTPDiffTitleDifferentLongWords = optional.Some(state)
}

// ComputeDNSExperimentFailure computes the DNSExperimentFailure field.
func (wa *WebAnalysis) ComputeDNSExperimentFailure(c *WebObservationsContainer) {

	for _, obs := range c.DNSLookupFailures {
		// make sure we have probe domain
		probeDomain := obs.DNSDomain.UnwrapOr("")
		if probeDomain == "" {
			continue
		}

		// make sure we have TH domain
		thDomain := obs.ControlDNSDomain.UnwrapOr("")
		if thDomain == "" {
			continue
		}

		// we only care about cases where we're resolving the same domain
		if probeDomain != thDomain {
			continue
		}

		// as documented, only include the system resolver
		if !utilsEngineIsGetaddrinfo(obs.DNSEngine) {
			continue
		}

		// skip cases where there's no DNS record for AAAA, which is a false positive
		//
		// in principle, this should not happen with getaddrinfo, but we add this
		// check nonetheless for robustness against this corner case
		if utilsDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
			continue
		}

		// only record the first failure
		//
		// we should only consider the first DNS lookup to be consistent with
		// what was previously returned by Web Connectivity v0.4
		wa.DNSExperimentFailure = obs.DNSLookupFailure
		return
	}
}

// ComputeDNSPossiblyNonexistingDomains computes the DNSPossiblyNonexistingDomains field.
func (wa *WebAnalysis) ComputeDNSPossiblyNonexistingDomains(c *WebObservationsContainer) {
	var state map[string]bool

	// first inspect the failures
	for _, obs := range c.DNSLookupFailures {
		// skip the comparison if we don't have enough information
		if obs.DNSLookupFailure.IsNone() || obs.ControlDNSLookupFailure.IsNone() {
			continue
		}

		// flip state from None to empty when we see the first couple of
		// (probe, th) failures allowing us to perform a comparison
		if state == nil {
			state = make(map[string]bool)
		}

		// assume the domain is set in both cases
		domain := obs.DNSDomain.Unwrap()
		runtimex.Assert(domain == obs.ControlDNSDomain.Unwrap(), "mismatch between domain names")

		// a domain is nonexisting if both the probe and the TH say so
		if obs.DNSLookupFailure.Unwrap() != netxlite.FailureDNSNXDOMAINError {
			continue
		}
		if obs.ControlDNSLookupFailure.Unwrap() != "dns_name_error" {
			continue
		}

		// set the state
		state[domain] = true
	}

	// then inspect the successes
	for _, obs := range c.DNSLookupSuccesses {
		// skip the comparison if we don't have enough information
		if obs.DNSLookupFailure.IsNone() && obs.ControlDNSLookupFailure.IsNone() {
			continue
		}

		// assume the domain is always set
		domain := obs.DNSDomain.Unwrap()

		// clear the state if the probe succeeded
		if !obs.DNSLookupFailure.IsNone() && obs.DNSLookupFailure.Unwrap() == "" {
			delete(state, domain)
			continue
		}

		// clear the state if the TH succeded
		if !obs.ControlDNSLookupFailure.IsNone() && obs.ControlDNSLookupFailure.Unwrap() == "" {
			runtimex.Assert(domain == obs.ControlDNSDomain.Unwrap(), "mismatch between domain names")
			delete(state, domain)
			continue
		}
	}

	// note that optional.Some constructs None if state is nil
	wa.DNSPossiblyNonexistingDomains = optional.Some(state)
}
