package minipipeline

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// AnalyzeWebObservations generates a [*WebAnalysis] from a [*WebObservationsContainer].
func AnalyzeWebObservations(container *WebObservationsContainer) *WebAnalysis {
	analysis := &WebAnalysis{}

	analysis.ComputeDNSExperimentFailure(container)
	analysis.ComputeDNSTransactionsWithBogons(container)
	analysis.ComputeDNSTransactionsWithUnexpectedFailures(container)
	analysis.ComputeDNSPossiblyInvalidAddrs(container)
	analysis.ComputeDNSPossiblyInvalidAddrsClassic(container)
	analysis.ComputeDNSPossiblyNonexistingDomains(container)

	analysis.ComputeTCPTransactionsWithUnexpectedTCPConnectFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedHTTPFailures(container)
	analysis.ComputeTCPTransactionsWithUnexplainedUnexpectedFailures(container)

	analysis.ComputeHTTPDiffBodyProportionFactor(container)
	analysis.ComputeHTTPDiffStatusCodeMatch(container)
	analysis.ComputeHTTPDiffUncommonHeadersIntersection(container)
	analysis.ComputeHTTPDiffTitleDifferentLongWords(container)
	analysis.ComputeHTTPFinalResponsesWithControl(container)
	analysis.ComputeHTTPFinalResponsesWithTLS(container)

	return analysis
}

// WebAnalysis summarizes the content of [*WebObservationsContainer].
//
// The zero value of this struct is ready to use.
//
// For optional fields, they are None (i.e., `null` in JSON, `nil` in Go) when the corresponding
// algorithm either didn't run or didn't encounter enough data to determine a non-None result. When
// they are not None, they can still be empty (e.g., `{}` in JSON and in Go). In the latter case,
// them being empty means we encountered good enough data to determine whether we needed to add
// something to such a field and decided not to. For example, DNSTransactionWithBogons being None
// means that there are no suitable transactions to inspect. It being empty, instead, means we
// have transactions to inspect but none of them contains bogons. In other words, most fields are
// three state and one should take this into account when performing data analysis.
type WebAnalysis struct {
	// DNSExperimentFailure is the first failure experienced by a getaddrinfo-like resolver.
	DNSExperimentFailure optional.Value[string]

	// DNSTransactionsWithBogons contains the list of DNS transactions containing bogons.
	DNSTransactionsWithBogons optional.Value[map[int64]bool]

	// DNSTransactionsWithUnexpectedFailures contains the DNS transaction IDs that
	// contain failures while the control measurement succeeded. Note that we don't
	// include DNS-over-HTTPS failures inside the list, because a DoH failure is
	// not related to the domain we're querying for.
	DNSTransactionsWithUnexpectedFailures optional.Value[map[int64]bool]

	// DNSPossiblyInvalidAddrs contains the addresses that are not valid for the
	// domain. An addres is valid for the domain if:
	//
	// 1. we can TLS handshake with the expected SNI using this address; or
	//
	// 2. the address was resolved by the TH; or
	//
	// 3. the address ASN belongs to the set of ASNs obtained by mapping
	// addresses resolved by the TH to their corresponding ASN.
	DNSPossiblyInvalidAddrs optional.Value[map[string]bool]

	// DNSPossiblyInvalidAddrsClassic is like DNSPossiblyInvalidAddrs but does
	// not use TLS to validate the IP addresses.
	DNSPossiblyInvalidAddrsClassic optional.Value[map[string]bool]

	// DNSPossiblyNonexistingDomains lists all the domains for which both
	// the probe and the TH failed to perform DNS lookups.
	DNSPossiblyNonexistingDomains optional.Value[map[string]bool]

	// HTTPDiffBodyProportionFactor is the body proportion factor.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffBodyProportionFactor optional.Value[float64]

	// HTTPDiffStatusCodeMatch returns whether the status code matches.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffStatusCodeMatch optional.Value[bool]

	// HTTPDiffTitleDifferentLongWords contains the words long 5+ characters that appear
	// in the probe's "final" response title or in the TH title but not in both.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffTitleDifferentLongWords optional.Value[map[string]bool]

	// HTTPDiffUncommonHeadersIntersection contains the uncommon headers intersection.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffUncommonHeadersIntersection optional.Value[map[string]bool]

	// HTTPFinalResponsesWithControl contains the transaction IDs of "final" responses (i.e.,
	// responses that are like 2xx, 4xx, or 5xx) for which we also have a valid HTTP control
	// measurement. Typically, we expect to have a single response that is final when
	// analyzing Web Connectivity LTE results.
	HTTPFinalResponsesWithControl optional.Value[map[int64]bool]

	// HTTPFinalResponsesWithTLS is like HTTPFinalResponses but only includes the
	// cases where we're using TLS to fetch the final response, and does not concern
	// itself with whether there's control data, because TLS suffices.
	HTTPFinalResponsesWithTLS optional.Value[map[int64]bool]

	// TCPTransactionsWithUnexpectedTCPConnectFailures contains the TCP transaction IDs that
	// contain TCP connect failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedTCPConnectFailures optional.Value[map[int64]bool]

	// TCPTransactionsWithUnexpectedTLSHandshakeFailures contains the TCP transaction IDs that
	// contain TLS handshake failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedTLSHandshakeFailures optional.Value[map[int64]bool]

	// TCPSTransactionsWithUnexpectedHTTPFailures contains the TCP transaction IDs that
	// contain HTTP failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedHTTPFailures optional.Value[map[int64]bool]

	// TCPTransactionsWithUnexplainedUnexpectedFailures contains the TCP transaction IDs for
	// which we cannot explain TCP or TLS failures with control information, but for which we
	// expect to see a success because the control's HTTP succeeded.
	TCPTransactionsWithUnexplainedUnexpectedFailures optional.Value[map[int64]bool]
}

func analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" &&
		obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
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
		if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
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

// ComputeDNSTransactionsWithBogons computes the DNSTransactionsWithBogons field.
func (wa *WebAnalysis) ComputeDNSTransactionsWithBogons(c *WebObservationsContainer) {
	// Implementation note: any bogon IP address resolved by a DoH service
	// is STILL suspicious since it SHOULD NOT happen. TODO(bassosimone): an
	// even better algorithm could possibly check whether also the TH has
	// observed bogon IP addrs and avoid flagging in such a case.
	//
	// See https://github.com/ooni/probe/issues/2274 for more information.

	// we cannot flip the state from None to empty until we inspect at least
	// a single successful DNS lookup transaction
	if len(c.DNSLookupSuccesses) <= 0 {
		return
	}
	state := make(map[int64]bool)

	for _, obs := range c.DNSLookupSuccesses {
		// do nothing if we don't know whether there's a bogon
		if obs.IPAddressBogon.IsNone() {
			continue
		}

		// do nothing if there is no bogon
		if !obs.IPAddressBogon.Unwrap() {
			continue
		}

		// update state
		if id := obs.DNSTransactionID.UnwrapOr(0); id > 0 {
			state[id] = true
		}
	}

	// note that optional.Some constructs None if state is nil
	wa.DNSTransactionsWithBogons = optional.Some(state)
}

func analysisDNSEngineIsDNSOverHTTPS(obs *WebObservation) bool {
	return obs.DNSEngine.UnwrapOr("") == "doh"
}

// ComputeDNSTransactionsWithUnexpectedFailures computes the DNSTransactionsWithUnexpectedFailures field.
func (wa *WebAnalysis) ComputeDNSTransactionsWithUnexpectedFailures(c *WebObservationsContainer) {
	// Implementation note: a DoH failure is not information about the URL we're
	// measuring but about the DoH service being blocked.
	//
	// See https://github.com/ooni/probe/issues/2274

	var state map[int64]bool

	for _, obs := range c.DNSLookupFailures {
		// skip cases where the engine is doh (see above comment)
		if analysisDNSEngineIsDNSOverHTTPS(obs) {
			continue
		}

		// skip cases where there's no DNS record for AAAA, which is a false positive
		if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
			continue
		}

		// TODO(bassosimone): if we set an IPv6 address as the resolver address, we
		// end up with false positive errors when there's no IPv6 support

		// skip cases with no control
		if obs.ControlDNSLookupFailure.IsNone() {
			continue
		}

		// flip from None to empty if we have seen at least one entry for
		// which we can compare to the control
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip cases where the control failed as well
		if obs.ControlDNSLookupFailure.Unwrap() != "" {
			continue
		}

		// update state
		if id := obs.DNSTransactionID.UnwrapOr(0); id > 0 {
			state[id] = true
		}
	}

	// note that optional.Some constructs None if state is nil
	wa.DNSTransactionsWithUnexpectedFailures = optional.Some(state)
}

// ComputeDNSPossiblyInvalidAddrs computes the DNSPossiblyInvalidAddrs field.
func (wa *WebAnalysis) ComputeDNSPossiblyInvalidAddrs(c *WebObservationsContainer) {
	// Implementation note: in the case in which DoH returned answers, here
	// it still feels okay to consider them. We should avoid flagging DoH
	// failures as measurement failures but if DoH returns us some unexpected
	// even-non-bogon addr, it seems worth flagging for now.
	//
	// See https://github.com/ooni/probe/issues/2274

	var state map[string]bool

	// pass 1: insert candidates into the state map
	for _, obs := range c.KnownTCPEndpoints {
		addr := obs.IPAddress.Unwrap()

		// skip the comparison if we don't have info about matching
		if obs.MatchWithControlIPAddress.IsNone() || obs.MatchWithControlIPAddressASN.IsNone() {
			continue
		}

		// flip state from None to empty when we see the first couple of
		// (probe, th) failures allowing us to perform a comparison
		if state == nil {
			state = make(map[string]bool)
		}

		// an address is suspicious if we have information regarding its potential
		// matching with TH info and we know it does not match
		if !obs.MatchWithControlIPAddress.Unwrap() && !obs.MatchWithControlIPAddressASN.Unwrap() {
			state[addr] = true
			continue
		}
	}

	// pass 2: remove IP addresses we could validate using TLS handshakes
	//
	// we need to perform this second step because the order with which we walk
	// through c.KnownTCPEndpoints is not fixed _and_ in any case, there is no
	// guarantee that we'll observe 80/tcp entries _before_ 443/tcp ones. So, by
	// applying this algorithm as a second step, we ensure that we're always
	// able to remove TLS-validate addresses from the "bad" set.
	for _, obs := range c.KnownTCPEndpoints {
		addr := obs.IPAddress.Unwrap()

		// we cannot do much if we don't have TLS handshake information
		if obs.TLSHandshakeFailure.IsNone() {
			continue
		}

		// we should not modify the state if the TLS handshake failed
		if obs.TLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		delete(state, addr)
	}

	// note that optional.Some constructs None if state is nil
	wa.DNSPossiblyInvalidAddrs = optional.Some(state)
}

// ComputeDNSPossiblyInvalidAddrsClassic computes the DNSPossiblyInvalidAddrsClassic field.
func (wa *WebAnalysis) ComputeDNSPossiblyInvalidAddrsClassic(c *WebObservationsContainer) {
	// Implementation note: in the case in which DoH returned answers, here
	// it still feels okay to consider them. We should avoid flagging DoH
	// failures as measurement failures but if DoH returns us some unexpected
	// even-non-bogon addr, it seems worth flagging for now.
	//
	// See https://github.com/ooni/probe/issues/2274

	var state map[string]bool

	for _, obs := range c.KnownTCPEndpoints {
		addr := obs.IPAddress.Unwrap()

		// skip the comparison if we don't have info about matching
		if obs.MatchWithControlIPAddress.IsNone() || obs.MatchWithControlIPAddressASN.IsNone() {
			continue
		}

		// flip state from None to empty when we see the first couple of
		// (probe, th) failures allowing us to perform a comparison
		if state == nil {
			state = make(map[string]bool)
		}

		// an address is suspicious if we have information regarding its potential
		// matching with TH info and we know it does not match
		if !obs.MatchWithControlIPAddress.Unwrap() && !obs.MatchWithControlIPAddressASN.Unwrap() {
			state[addr] = true
			continue
		}
	}

	// note that optional.Some constructs None if state is nil
	wa.DNSPossiblyInvalidAddrsClassic = optional.Some(state)
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

func analysisTCPConnectFailureSeemsMisconfiguredIPv6(obs *WebObservation) bool {
	switch obs.TCPConnectFailure.UnwrapOr("") {
	case netxlite.FailureNetworkUnreachable, netxlite.FailureHostUnreachable:
		isv6, err := netxlite.IsIPv6(obs.IPAddress.UnwrapOr(""))
		return err == nil && isv6

	default: // includes the case of missing TCPConnectFailure
		return false
	}
}

// ComputeTCPTransactionsWithUnexpectedTCPConnectFailures computes the TCPTransactionsWithUnexpectedTCPConnectFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedTCPConnectFailures(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// we cannot do anything unless we have both records
		if obs.TCPConnectFailure.IsNone() || obs.ControlTCPConnectFailure.IsNone() {
			continue
		}

		// flip state from None to empty once we have seen the first
		// suitable set of measurement/control pairs
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip cases with no failures
		if obs.TCPConnectFailure.Unwrap() == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTCPConnectFailure.Unwrap() != "" {
			continue
		}

		// skip cases where the root cause could be a misconfigured IPv6 stack
		if analysisTCPConnectFailureSeemsMisconfiguredIPv6(obs) {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	// note that optional.Some constructs None if state is nil
	wa.TCPTransactionsWithUnexpectedTCPConnectFailures = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures computes the TCPTransactionsWithUnexpectedTLSHandshakeFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// we cannot do anything unless we have both records
		if obs.TLSHandshakeFailure.IsNone() || obs.ControlTLSHandshakeFailure.IsNone() {
			continue
		}

		// flip state from None to empty once we have seen the first
		// suitable set of measurement/control pairs
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip cases with no failures
		if obs.TLSHandshakeFailure.Unwrap() == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	// note that optional.Some constructs None if state is nil
	wa.TCPTransactionsWithUnexpectedTLSHandshakeFailures = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexpectedHTTPFailures computes the TCPTransactionsWithUnexpectedHTTPFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedHTTPFailures(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// we cannot do anything unless we have both records
		if obs.HTTPFailure.IsNone() || obs.ControlHTTPFailure.IsNone() {
			continue
		}

		// flip state from None to empty once we have seen the first
		// suitable set of measurement/control pairs
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip cases with no failures
		if obs.HTTPFailure.Unwrap() == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlHTTPFailure.Unwrap() != "" {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	// note that optional.Some constructs None if state is nil
	wa.TCPTransactionsWithUnexpectedHTTPFailures = optional.Some(state)
}

// ComputeHTTPDiffBodyProportionFactor computes the HTTPDiffBodyProportionFactor field.
func (wa *WebAnalysis) ComputeHTTPDiffBodyProportionFactor(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// we should only perform the comparison for a final response
		if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
			continue
		}

		// we need a valid body length and the body must not be truncated
		measurement := obs.HTTPResponseBodyLength.UnwrapOr(0)
		if measurement <= 0 || obs.HTTPResponseBodyIsTruncated.UnwrapOr(true) {
			continue
		}

		// we also need a valid control body length
		control := obs.ControlHTTPResponseBodyLength.UnwrapOr(0)
		if control <= 0 {
			continue
		}

		// compute the body proportion factor
		var proportion float64
		if measurement >= control {
			proportion = float64(control) / float64(measurement)
		} else {
			proportion = float64(measurement) / float64(control)
		}

		// update state
		wa.HTTPDiffBodyProportionFactor = optional.Some(proportion)

		// Implementation note: we only process the first observation that matches.
		//
		// This is fine(TM) as long as we have a single "final" response.
		break
	}
}

// ComputeHTTPDiffStatusCodeMatch computes the HTTPDiffStatusCodeMatch field.
func (wa *WebAnalysis) ComputeHTTPDiffStatusCodeMatch(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
		// we should only perform the comparison for a final response
		if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
			continue
		}

		// we need a positive status code for both
		measurement := obs.HTTPResponseStatusCode.UnwrapOr(0)
		if measurement <= 0 {
			continue
		}
		control := obs.ControlHTTPResponseStatusCode.UnwrapOr(0)
		if control <= 0 {
			continue
		}

		// compute whether there's a match including caveats
		good := control == measurement
		if !good && control/100 != 2 {
			// Avoid comparison if it seems the TH failed _and_ the two
			// status codes are not equal. Originally, this algorithm was
			// https://github.com/measurement-kit/measurement-kit/blob/b55fbecb205be62c736249b689df0c45ae342804/src/libmeasurement_kit/ooni/web_connectivity.cpp#L60
			// and excluded the case where the TH failed with 5xx.
			//
			// Then, we discovered when implementing websteps a bunch
			// of control failure modes that suggested to be more
			// cautious. See https://github.com/bassosimone/websteps-illustrated/blob/632f27443ab9d94fb05efcf5e0b0c1ce190221e2/internal/engine/experiment/websteps/analysisweb.go#L137.
			//
			// However, it seems a bit retarded to avoid comparison
			// when both the TH and the probe failed equally. See
			// https://github.com/ooni/probe/issues/2287, which refers
			// to a measurement where both the probe and the TH fail
			// with 404, but we fail to say "status_code_match = true".
			//
			// See https://explorer.ooni.org/measurement/20220911T203447Z_webconnectivity_IT_30722_n1_YDZQZOHAziEJk6o9?input=http%3A%2F%2Fwww.webbox.com%2Findex.php
			// for a measurement where this was fixed.
			return
		}

		// update state
		wa.HTTPDiffStatusCodeMatch = optional.Some(good)

		// Implementation note: we only process the first observation that matches.
		//
		// This is fine(TM) as long as we have a single "final" request.
		break
	}
}

var analysisCommonHeaders = map[string]bool{
	"date":                      true,
	"content-type":              true,
	"server":                    true,
	"cache-control":             true,
	"vary":                      true,
	"set-cookie":                true,
	"location":                  true,
	"expires":                   true,
	"x-powered-by":              true,
	"content-encoding":          true,
	"last-modified":             true,
	"accept-ranges":             true,
	"pragma":                    true,
	"x-frame-options":           true,
	"etag":                      true,
	"x-content-type-options":    true,
	"age":                       true,
	"via":                       true,
	"p3p":                       true,
	"x-xss-protection":          true,
	"content-language":          true,
	"cf-ray":                    true,
	"strict-transport-security": true,
	"link":                      true,
	"x-varnish":                 true,
}

// ComputeHTTPDiffUncommonHeadersIntersection computes the HTTPDiffUncommonHeadersIntersection field.
func (wa *WebAnalysis) ComputeHTTPDiffUncommonHeadersIntersection(c *WebObservationsContainer) {
	state := make(map[string]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// we should only perform the comparison for a final response
		if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
			continue
		}

		// We should only perform the comparison if we have valid control data. Because
		// the headers could legitimately be empty, let's use the status code here.
		if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
			continue
		}

		// Implementation note: here we need to continue running when either
		// headers are empty in order to produce an empty intersection. If we'd stop
		// after noticing that either dictionary is empty, we'd produce a nil
		// analysis result, which causes QA differences with v0.4.
		measurement := obs.HTTPResponseHeadersKeys.UnwrapOr(nil)
		control := obs.ControlHTTPResponseHeadersKeys.UnwrapOr(nil)

		const (
			byProbe = 1 << iota
			byTH
		)

		matching := make(map[string]int64)
		for key := range measurement {
			key = strings.ToLower(key)
			if _, ok := analysisCommonHeaders[key]; !ok {
				matching[key] |= byProbe
			}
		}

		for key := range control {
			key = strings.ToLower(key)
			if _, ok := analysisCommonHeaders[key]; !ok {
				matching[key] |= byTH
			}
		}

		// compute the intersection of uncommon headers
		for key, value := range matching {
			if (value & (byProbe | byTH)) == (byProbe | byTH) {
				state[key] = true
			}
		}

		// Implementation note: we only process the first observation that matches.
		//
		// This is fine(TM) as long as we have a single "final" request.
		wa.HTTPDiffUncommonHeadersIntersection = optional.Some(state)
		break
	}
}

// ComputeHTTPDiffTitleDifferentLongWords computes the HTTPDiffTitleDifferentLongWords field.
func (wa *WebAnalysis) ComputeHTTPDiffTitleDifferentLongWords(c *WebObservationsContainer) {
	state := make(map[string]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// we should only perform the comparison for a final response
		if !obs.HTTPResponseIsFinal.UnwrapOr(false) {
			continue
		}

		// We should only perform the comparison if we have valid control data. Because
		// the title could legitimately be empty, let's use the status code here.
		if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
			continue
		}

		measurement := obs.HTTPResponseTitle.UnwrapOr("")
		control := obs.ControlHTTPResponseTitle.UnwrapOr("")

		const (
			byProbe = 1 << iota
			byTH
		)

		// Implementation note
		//
		// We don't consider to match words that are shorter than 5
		// characters (5 is the average word length for english)
		//
		// The original implementation considered the word order but
		// considering different languages it seems we could have less
		// false positives by ignoring the word order.
		words := make(map[string]int64)
		const minWordLength = 5
		for _, word := range strings.Split(measurement, " ") {
			if len(word) >= minWordLength {
				words[strings.ToLower(word)] |= byProbe
			}
		}
		for _, word := range strings.Split(control, " ") {
			if len(word) >= minWordLength {
				words[strings.ToLower(word)] |= byTH
			}
		}

		// compute the list of long words that do not appear in both titles
		for word, score := range words {
			if (score & (byProbe | byTH)) != (byProbe | byTH) {
				state[word] = true
			}
		}

		// Implementation note: we only process the first observation that matches.
		//
		// This is fine(TM) as long as we have a single "final" request.
		wa.HTTPDiffTitleDifferentLongWords = optional.Some(state)
		break
	}
}

// ComputeHTTPFinalResponsesWithControl computes the HTTPFinalResponses field.
func (wa *WebAnalysis) ComputeHTTPFinalResponsesWithControl(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// skip this entry if we don't know the transaction ID
		txid := obs.EndpointTransactionID.UnwrapOr(0)
		if txid <= 0 {
			continue
		}

		// skip this entry if it's not final
		isFinal := obs.HTTPResponseIsFinal.UnwrapOr(false)
		if !isFinal {
			continue
		}

		// skip this entry if don't have control information
		if obs.ControlHTTPFailure.IsNone() {
			continue
		}

		// flip state from None to empty when we have seen the first final
		// response for which we have valid control info
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip in case the HTTP control failed
		if obs.ControlHTTPFailure.Unwrap() != "" {
			continue
		}

		state[txid] = true
	}

	// note that optional.Some constructs None if state is nil
	wa.HTTPFinalResponsesWithControl = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexplainedUnexpectedFailures computes the TCPTransactionsWithUnexplainedUnexpectedFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexplainedUnexpectedFailures(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// obtain the transaction ID
		txid := obs.EndpointTransactionID.UnwrapOr(0)
		if txid <= 0 {
			continue
		}

		// to execute the algorithm we must have the reasonable expectation of
		// success, which we have iff the control succeeded.
		if obs.ControlHTTPFailure.IsNone() || obs.ControlHTTPFailure.Unwrap() != "" {
			continue
		}

		// flip state from None to empty when we have a reasonable
		// expectation of success as explained above
		if state == nil {
			state = make(map[int64]bool)
		}

		// if we have a TCP connect measurement, the measurement failed, and we don't have
		// a corresponding control measurement, we cannot explain this failure using the control
		//
		// while doing this, deal with misconfigured-IPv6 false positives
		if !obs.TCPConnectFailure.IsNone() && obs.TCPConnectFailure.Unwrap() != "" &&
			!analysisTCPConnectFailureSeemsMisconfiguredIPv6(obs) &&
			obs.ControlTCPConnectFailure.IsNone() {
			state[txid] = true
			continue
		}

		// if we have a TLS handshake measurement, the measurement failed, and we don't have
		// a corresponding control measurement, we cannot explain this failure using the control
		if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() != "" &&
			obs.ControlTLSHandshakeFailure.IsNone() {
			state[txid] = true
			continue
		}
	}

	// note that optional.Some constructs None if state is nil
	wa.TCPTransactionsWithUnexplainedUnexpectedFailures = optional.Some(state)
}

// ComputeHTTPFinalResponsesWithTLS computes the HTTPFinalResponsesWithTLS field.
func (wa *WebAnalysis) ComputeHTTPFinalResponsesWithTLS(c *WebObservationsContainer) {
	var state map[int64]bool

	for _, obs := range c.KnownTCPEndpoints {
		// skip this entry if we don't know the transaction ID
		txid := obs.EndpointTransactionID.UnwrapOr(0)
		if txid <= 0 {
			continue
		}

		// skip this entry if it's not final
		isFinal := obs.HTTPResponseIsFinal.UnwrapOr(false)
		if !isFinal {
			continue
		}

		// skip this entry if we didn't try a TLS handshake
		if obs.TLSHandshakeFailure.IsNone() {
			continue
		}

		// flip the state from None to empty when we have an endpoint
		// for which we attempted a TLS handshake
		if state == nil {
			state = make(map[int64]bool)
		}

		// skip in case the TLS handshake failed
		if obs.TLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		state[txid] = true
	}

	// note that optional.Some constructs None if state is nil
	wa.HTTPFinalResponsesWithTLS = optional.Some(state)
}
