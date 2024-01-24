package minipipeline

import (
	"sort"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewLinearWebAnalysis constructs a slice containing all the observations
// contained inside the given [*WebObservationsContainer].
//
// We sort the returned list as follows:
//
// 1. by descending TagDepth;
//
// 2. with TagDepth being equal, by descending [WebObservationType];
//
// 3. with [WebObservationType] being equal, by ascending failure string;
//
// This means that you divide the list in groups like this:
//
//	+------------+------------+------------+------------+
//	| TagDepth=3 | TagDepth=2 | TagDepth=1 | TagDepth=0 |
//	+------------+------------+------------+------------+
//
// Where TagDepth=3 is the last redirect and TagDepth=0 is the initial request.
//
// Each group is further divided as follows:
//
//	+------+-----+-----+-----+
//	| HTTP | TLS | TCP | DNS |
//	+------+-----+-----+-----+
//
// Where each group may be empty. The first non-empty group is about the
// operation that failed for the current TagDepth.
//
// Within each group, successes sort before failures because the empty
// string has priority over non-empty strings.
//
// So, when walking the list from index 0 to index N, you encounter the
// latest redirects first, you observe the more complex operations first,
// and you see errors before failures.
func NewLinearWebAnalysis(input *WebObservationsContainer) (output []*WebObservation) {
	// fill in all the observations
	output = append(output, input.DNSLookupFailures...)
	output = append(output, input.DNSLookupSuccesses...)
	for _, entry := range input.KnownTCPEndpoints {
		output = append(output, entry)
	}

	// sort in descending order
	sort.SliceStable(output, func(i, j int) bool {
		left, right := output[i], output[j]

		// We use -1 as the default value such that observations with undefined
		// TagDepth sort at the end of the generated list.
		if left.TagDepth.UnwrapOr(-1) > right.TagDepth.UnwrapOr(-1) {
			return true
		} else if left.TagDepth.UnwrapOr(-1) < right.TagDepth.UnwrapOr(-1) {
			return false
		}

		if left.Type > right.Type {
			return true
		} else if left.Type < right.Type {
			return false
		}

		// We use an nonempty failure value so that observations with undefined
		// failures sort at the end of the group within the list.
		const defaultFailureValue = "unknown_failure"
		if left.Failure.UnwrapOr(defaultFailureValue) < right.Failure.UnwrapOr(defaultFailureValue) {
			return true
		} else if left.Failure.UnwrapOr(defaultFailureValue) > right.Failure.UnwrapOr(defaultFailureValue) {
			return false
		}

		// This is undocumented but important. KnownTCPEndpoints is a map and iterating
		// it causes the order to change from test run to test run. To ensure there's
		// stable and comparable sorting of the results, we need to introduce an extra
		// rule allowing us to choose between two distinct endpoint observations.
		//
		// (While DNS observations using UDP or HTTPS may use the same transaction ID
		// for some time until we manage to split transactions[1], we know for sure
		// that endpoints are going to use always distinct transactions.)
		//
		// .. [1] https://github.com/ooni/probe/issues/2624
		return left.TransactionID > right.TransactionID
	})

	return
}

// AnalyzeWebObservationsWithoutLinearAnalysis generates a [*WebAnalysis] from a [*WebObservationsContainer]
// but avoids calling [NewLinearyAnalysis] to generate a linear analysis.
func AnalyzeWebObservationsWithoutLinearAnalysis(
	lookupper model.GeoIPASNLookupper, container *WebObservationsContainer) *WebAnalysis {
	analysis := &WebAnalysis{
		ControlExpectations: container.ControlExpectations,
	}

	analysis.dnsComputeSuccessMetrics(lookupper, container)
	analysis.dnsComputeSuccessMetricsClassic(lookupper, container)
	analysis.dnsComputeFailureMetrics(container)

	analysis.tcpComputeMetrics(container)
	analysis.tlsComputeMetrics(container)
	analysis.httpComputeFailureMetrics(container)
	analysis.httpComputeFinalResponseMetrics(container)

	return analysis
}

// AnalyzeWebObservationsWithLinearAnalysis generates a [*WebAnalysis] from a [*WebObservationsContainer]
// and ensures we also use [NewLinearyAnalysis] to generate a linear analysis.
func AnalyzeWebObservationsWithLinearAnalysis(
	lookupper model.GeoIPASNLookupper, container *WebObservationsContainer) *WebAnalysis {
	analysis := AnalyzeWebObservationsWithoutLinearAnalysis(lookupper, container)
	analysis.Linear = NewLinearWebAnalysis(container)
	return analysis
}

// WebAnalysis summarizes the content of [*WebObservationsContainer].
//
// The zero value of this struct is ready to use.
type WebAnalysis struct {
	// ControlExpectations summarizes the expectations we have for the control. You should
	// use this field to determine (a) whether unexplained failures are expected or unexpected
	// and (b) whether we expect to see any resolved IP address and (c) possibly more flags.
	ControlExpectations optional.Value[*WebObservationsControlExpectations]

	// DNSLookupSuccess contains all DNS lookups that were successful and for which
	// we potentially have control information (i.e., in the 0-th redirect).
	DNSLookupSuccess Set[int64]

	// DNSLookupSuccessWithInvalidAddresses contains DNS transactions with invalid IP addresses by
	// taking into account control info, bogons, and TLS handshakes.
	DNSLookupSuccessWithInvalidAddresses Set[int64]

	// DNSLookupSuccessWithValidAddress contains DNS transactions with valid IP addresses.
	DNSLookupSuccessWithValidAddress Set[int64]

	// DNSLookupSuccessWithBogonAddresses contains DNS transactions with bogon IP addresses.
	DNSLookupSuccessWithBogonAddresses Set[int64]

	// DNSLookupSuccessWithInvalidAddressesClassic is like DNSLookupInvalid but the algorithm is more relaxed
	// to be compatible with Web Connectivity v0.4's behavior.
	DNSLookupSuccessWithInvalidAddressesClassic Set[int64]

	// DNSLookupSuccessWithValidAddressClassic contains DNS transactions with valid IP addresses.
	DNSLookupSuccessWithValidAddressClassic Set[int64]

	// DNSLookupUnexpectedFailure contains DNS transactions with unexpected failures.
	DNSLookupUnexpectedFailure Set[int64]

	// DNSLookupUnexplainedFailure contains DNS transactions with unexplained failures (i.e.,
	// failures for which there's no corresponding control information).
	DNSLookupUnexplainedFailure Set[int64]

	// DNSExperimentFailure is the first failure experienced by any resolver
	// before hitting redirects (i.e., when TagDepth==0).
	DNSExperimentFailure optional.Value[string]

	// DNSLookupExpectedFailure contains DNS transactions with expected failures, i.e.,
	// failures that are consistent for the probe and the TH.
	DNSLookupExpectedFailure Set[int64]

	// DNSLookupExpectedSuccess contains DNS transactions with expected successes.
	DNSLookupExpectedSuccess Set[int64]

	// TCPConnectExpectedFailure contains TCP connect transactions that failed
	// consistently for the probe and the test helper.
	TCPConnectExpectedFailure Set[int64]

	// TCPConnectUnexpectedFailure contains TCP endpoint transactions with unexpected failures.
	TCPConnectUnexpectedFailure Set[int64]

	// TCPConnectUnexpectedFailureDuringWebFetch contains TCP endpoint transactions with unexpected failures
	// while performing a web fetch, as opposed to checking for connectivity.
	TCPConnectUnexpectedFailureDuringWebFetch Set[int64]

	// TCPConnectUnexpectedFailureDuringConnectivityCheck contains TCP endpoint transactions with unexpected failures
	// while checking for connectivity, as opposed to fetching a webpage.
	TCPConnectUnexpectedFailureDuringConnectivityCheck Set[int64]

	// TCPConnectUnexplainedFailure contains failures occurring during redirects, i.e.,
	// failures for which there's no corresponding control info.
	TCPConnectUnexplainedFailure Set[int64]

	// TCPConnectUnexplainedFailureDuringWebFetch contains failures occurring during redirects
	// while performing a web fetch, as opposed to checking for connectivity.
	TCPConnectUnexplainedFailureDuringWebFetch Set[int64]

	// TCPConnectUnexplainedFailureDuringConnectivityCheck contains failures occurring during redirects
	// while checking for connectivity, as opposed to fetching a webpage.
	TCPConnectUnexplainedFailureDuringConnectivityCheck Set[int64]

	// TLSHandshakeExpectedFailure contains TLS endpoint transactions that failed
	// consistently for the probe and the test helper.
	TLSHandshakeExpectedFailure Set[int64]

	// TLSHandshakeUnexpectedFailure contains TLS endpoint transactions with unexpected failures.
	TLSHandshakeUnexpectedFailure Set[int64]

	// TLSHandshakeUnexpectedFailureDuringWebFetch contains TLS endpoint transactions with unexpected failures.
	// while performing a web fetch, as opposed to checking for connectivity.
	TLSHandshakeUnexpectedFailureDuringWebFetch Set[int64]

	// TLSHandshakeUnexpectedFailureDuringConnectivityCheck contains TLS endpoint transactions with unexpected failures.
	// while checking for connectivity, as opposed to fetching a webpage.
	TLSHandshakeUnexpectedFailureDuringConnectivityCheck Set[int64]

	// TLSHandshakeUnexplainedFailure contains failures occurring during redirects, i.e.,
	// failures for which there's no corresponding control info.
	TLSHandshakeUnexplainedFailure Set[int64]

	// TLSHandshakeUnexplainedFailureDuringWebFetch  contains failures occurring during redirects
	// while performing a web fetch, as opposed to checking for connectivity.
	TLSHandshakeUnexplainedFailureDuringWebFetch Set[int64]

	// TLSHandshakeUnexplainedFailureDuringConnectivityCheck contains failures occurring during redirects
	// while checking for connectivity, as opposed to fetching a webpage.
	TLSHandshakeUnexplainedFailureDuringConnectivityCheck Set[int64]

	// HTTPRoundTripUnexpectedFailure contains HTTP endpoint transactions with unexpected failures.
	HTTPRoundTripUnexpectedFailure Set[int64]

	// HTTPRoundTripUnexplainedFailure contains failures occurring during redirects, i.e.,
	// failures for which there's no corresponding control info.
	HTTPRoundTripUnexplainedFailure Set[int64]

	// HTTPFinalResponseSuccessTLSWithoutControl contains the ID of the final response
	// transaction when the final response succeeded without control and with TLS.
	HTTPFinalResponseSuccessTLSWithoutControl optional.Value[int64]

	// HTTPFinalResponseSuccessTLSWithControl contains the ID of the final response
	// transaction when the final response succeeded with control and with TLS.
	HTTPFinalResponseSuccessTLSWithControl optional.Value[int64]

	// HTTPFinalResponseSuccessTCPWithoutControl contains the ID of the final response
	// transaction when the final response succeeded without control and with TCP.
	HTTPFinalResponseSuccessTCPWithoutControl optional.Value[int64]

	// HTTPFinalResponseSuccessTCPWithControl contains the ID of the final response
	// transaction when the final response succeeded with control and with TCP.
	HTTPFinalResponseSuccessTCPWithControl optional.Value[int64]

	// HTTPFinalResponseDiffBodyProportionFactor is the body proportion factor.
	HTTPFinalResponseDiffBodyProportionFactor optional.Value[float64]

	// HTTPFinalResponseDiffStatusCodeMatch returns whether the status code matches.
	HTTPFinalResponseDiffStatusCodeMatch optional.Value[bool]

	// HTTPFinalResponseDiffTitleDifferentLongWords contains the words long 5+ characters that appear
	// in the probe's "final" response title or in the TH title but not in both.
	HTTPFinalResponseDiffTitleDifferentLongWords optional.Value[map[string]bool]

	// HTTPFinalResponseDiffUncommonHeadersIntersection contains the uncommon headers intersection.
	HTTPFinalResponseDiffUncommonHeadersIntersection optional.Value[map[string]bool]

	// Linear contains the linear analysis. We only fill this field when using
	// the [AnalyzeWebObservationsWithLinearAnalysis] constructor.
	Linear []*WebObservation
}

func (wa *WebAnalysis) dnsComputeSuccessMetrics(
	lookupper model.GeoIPASNLookupper, c *WebObservationsContainer) {
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

		// track all the 0-th redirect lookups that succeded which helps to
		// determine whether the probe UNEXPECTEDLY resolved any address.
		wa.DNSLookupSuccess.Add(obs.DNSTransactionID.Unwrap())

		// if there's a bogon, register it and continue processing.
		if !obs.IPAddressBogon.IsNone() && obs.IPAddressBogon.Unwrap() {
			wa.DNSLookupSuccessWithBogonAddresses.Add(obs.DNSTransactionID.Unwrap())
			// fallthrough: bogons are legitimate if the website DNS is actually misconfigured
			// so we determine bogons status and invalid addresses separately
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
			wa.DNSLookupSuccessWithValidAddress.Add(obs.DNSTransactionID.Unwrap())
			continue
		}

		// this lookup is good if there is ASN intersection
		if DNSDiffFindCommonASNsIntersection(lookupper, measurement, control).Len() > 0 {
			wa.DNSLookupSuccessWithValidAddress.Add(obs.DNSTransactionID.Unwrap())
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
		wa.DNSLookupSuccessWithValidAddress.Add(txid)
	}
}

func (wa *WebAnalysis) dnsComputeSuccessMetricsClassic(
	lookupper model.GeoIPASNLookupper, c *WebObservationsContainer) {
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
			wa.DNSLookupSuccessWithValidAddressClassic.Add(obs.DNSTransactionID.Unwrap())
			continue
		}

		// this lookup is good if there is ASN intersection
		if DNSDiffFindCommonASNsIntersection(lookupper, measurement, control).Len() > 0 {
			wa.DNSLookupSuccessWithValidAddressClassic.Add(obs.DNSTransactionID.Unwrap())
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

		// lookups once we started following redirects does not have corresponding
		// control information, so failures end up being unexplained
		if obs.TagDepth.IsNone() || obs.TagDepth.Unwrap() != 0 {
			if obs.DNSLookupFailure.Unwrap() != "" {
				wa.DNSLookupUnexplainedFailure.Add(obs.DNSTransactionID.Unwrap())
				continue
			}
			continue
		}

		// honor the DNSExperimentFailure by assigning the first
		// probe error that we see with depth==0
		if obs.DNSLookupFailure.Unwrap() != "" && wa.DNSExperimentFailure.IsNone() {
			wa.DNSExperimentFailure = obs.DNSLookupFailure
			// fallthrough
		}

		// handle the case where there's no control
		if obs.ControlDNSLookupFailure.IsNone() {
			continue
		}

		// Handle the case where both failed.
		//
		// See https://github.com/ooni/probe/issues/2290 for further
		// documentation about the issue we're solving here.
		//
		// It would be tempting to check specifically for NXDOMAIN here, but we
		// know it is problematic do that. In fact, on Android the getaddrinfo
		// resolver always returns EAI_NODATA on error, regardless of the actual
		// error that may have occurred in the Android DNS backend.
		//
		// See https://github.com/ooni/probe/issues/2029 for more information
		// on Android's getaddrinfo behavior.
		if obs.DNSLookupFailure.Unwrap() != "" && obs.ControlDNSLookupFailure.Unwrap() != "" {
			wa.DNSLookupExpectedFailure.Add(obs.DNSTransactionID.Unwrap())
			continue
		}

		// handle the case where only the control failed
		if obs.ControlDNSLookupFailure.Unwrap() != "" {
			continue
		}

		// handle the case where only the probe failed
		if failure := obs.DNSLookupFailure.Unwrap(); failure != "" {
			// When the probe says dns_no_answer the control would otherwise say that
			// we have resolved zero IP addresses for historical reasons. In such a case,
			// let's pretend also the control returned dns_no_answer.
			if failure == netxlite.FailureDNSNoAnswer {
				if !obs.ControlDNSResolvedAddrs.IsNone() && obs.ControlDNSResolvedAddrs.Unwrap().Len() <= 0 {
					wa.DNSLookupExpectedFailure.Add(obs.DNSTransactionID.Unwrap())
					continue
				}
			}
			wa.DNSLookupUnexpectedFailure.Add(obs.DNSTransactionID.Unwrap())
			continue
		}

		// handle the case where both succeed
		wa.DNSLookupExpectedSuccess.Add(obs.DNSTransactionID.Unwrap())
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
		//
		// See https://explorer.ooni.org/measurement/20220911T105037Z_webconnectivity_IT_30722_n1_ruzuQ219SmIO9SrT?input=https://doh.centraleu.pi-dns.com/dns-query?dns=q80BAAABAAAAAAAAA3d3dwdleGFtcGxlA2NvbQAAAQAB
		// for an example measurement with this behavior.
		//
		// See also https://github.com/ooni/probe/issues/2299.
		if obs.TCPConnectFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
			wa.TCPConnectExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
			continue
		}

		// handle the case where the control fails
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
		//
		// See https://github.com/ooni/probe/issues/2300.
		if obs.TLSHandshakeFailure.Unwrap() != "" && obs.ControlTLSHandshakeFailure.Unwrap() != "" {
			wa.TLSHandshakeExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
			continue
		}

		// handle the case where the control fails
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
			if obs.HTTPFailure.Unwrap() != "" {
				wa.HTTPRoundTripUnexplainedFailure.Add(obs.EndpointTransactionID.Unwrap())
				continue
			}
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
		wa.HTTPFinalResponseSuccessTCPWithoutControl = obs.EndpointTransactionID
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

// httpDiffBodyProportionFactor computes the body proportion factor.
//
// The return value--used for testing--is zero on success and negative in case of failure.
func (wa *WebAnalysis) httpDiffBodyProportionFactor(obs *WebObservation) int64 {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return -1
	}

	// we need a valid body length and the body must not be truncated
	measurement := obs.HTTPResponseBodyLength.UnwrapOr(0)
	if measurement <= 0 {
		return -2
	}
	if obs.HTTPResponseBodyIsTruncated.UnwrapOr(true) {
		return -3
	}

	// we also need a valid control body length
	control := obs.ControlHTTPResponseBodyLength.UnwrapOr(0)
	if control <= 0 {
		return -4
	}

	// compute the body proportion factor and update the state
	proportion := ComputeHTTPDiffBodyProportionFactor(measurement, control)
	wa.HTTPFinalResponseDiffBodyProportionFactor = optional.Some(proportion)
	return 0
}

// httpDiffStatusCodeMatch computes whether the status code matches.
//
// The return value--used for testing--is zero on success and negative in case of failure.
func (wa *WebAnalysis) httpDiffStatusCodeMatch(obs *WebObservation) int64 {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return -1
	}

	// we need a positive status code for both
	measurement := obs.HTTPResponseStatusCode.UnwrapOr(0)
	if measurement <= 0 {
		return -2
	}
	control := obs.ControlHTTPResponseStatusCode.UnwrapOr(0)
	if control <= 0 {
		return -3
	}

	// update state
	wa.HTTPFinalResponseDiffStatusCodeMatch = ComputeHTTPDiffStatusCodeMatch(measurement, control)
	return 0
}

// httpDiffUncommonHeadersIntersection computes the uncommon headers intersection.
//
// The return value--used for testing--is negative in case of failure and zero or positive otherwise.
func (wa *WebAnalysis) httpDiffUncommonHeadersIntersection(obs *WebObservation) int64 {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return -1
	}

	// We should only perform the comparison if we have valid control data. Because
	// the headers could legitimately be empty, let's use the status code here.
	if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
		return -2
	}

	// Implementation note: here we need to continue running when either
	// headers are empty in order to produce an empty intersection. If we'd stop
	// after noticing that either dictionary is empty, we'd produce a nil
	// analysis result, which causes QA differences with v0.4.
	measurement := obs.HTTPResponseHeadersKeys.UnwrapOr(nil)
	control := obs.ControlHTTPResponseHeadersKeys.UnwrapOr(nil)

	state := ComputeHTTPDiffUncommonHeadersIntersection(measurement, control)
	wa.HTTPFinalResponseDiffUncommonHeadersIntersection = optional.Some(state)
	return int64(len(state))
}

// httpDiffTitleDifferentLongWords computes the different long words.
//
// The return value--used for testing--is negative in case of failure and zero or positive otherwise.
func (wa *WebAnalysis) httpDiffTitleDifferentLongWords(obs *WebObservation) int64 {
	// we should only perform the comparison for a final response
	if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
		return -1
	}

	// We should only perform the comparison if we have valid control data. Because
	// the title could legitimately be empty, let's use the status code here.
	if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
		return -2
	}

	measurement := obs.HTTPResponseTitle.UnwrapOr("")
	control := obs.ControlHTTPResponseTitle.UnwrapOr("")

	state := ComputeHTTPDiffTitleDifferentLongWords(measurement, control)

	wa.HTTPFinalResponseDiffTitleDifferentLongWords = optional.Some(state)
	return int64(len(state))
}
