package minipipeline

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// WebAnalysis summarizes the content of [*WebObservationsContainer].
//
// The zero value of this struct is ready to use.
type WebAnalysis struct {
	// DNSExperimentFailure is the first failure experienced by a getaddrinfo-like resolver.
	DNSExperimentFailure optional.Value[string]

	// DNSTransactionsWithBogons contains the list of DNS transactions containing bogons.
	DNSTransactionsWithBogons optional.Value[map[int64]bool]

	// DNSTransactionsWithUnexpectedFailures contains the DNS transaction IDs that
	// contain failures while the control measurement succeeded. Note that we don't
	// include DNS-over-HTTPS failures inside this value.
	DNSTransactionsWithUnexpectedFailures optional.Value[map[int64]bool]

	// DNSPossiblyInvalidAddrs contains the addresses that are not valid for the
	// domain. An addres is valid for the domain if:
	//
	// 1. we can TLS handshake with the expected SNI using this address;
	//
	// 2. the address was resolved by the TH;
	//
	// 3. the address ASN was also observed by TH.
	DNSPossiblyInvalidAddrs optional.Value[map[string]bool]

	// HTTPDiffBodyProportionFactor is the body proportion factor.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffBodyProportionFactor optional.Value[float64]

	// HTTPDiffStatusCodeMatch returns whether the status code match modulo some
	// false positive cases that causes this value to remain null.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffStatusCodeMatch optional.Value[bool]

	// HTTPDiffTitleDifferentLongWords contains the words longer than 4 characters
	// that appear in the probe's "final" response title but don't appear in the TH title.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffTitleDifferentLongWords optional.Value[map[string]bool]

	// HTTPDiffUncommonHeadersIntersection contains the uncommon headers intersection.
	//
	// The generation algorithm assumes there's a single "final" response.
	HTTPDiffUncommonHeadersIntersection optional.Value[map[string]bool]

	// TCPTransactionsWithUnexpectedTCPConnectFailures contains the TCP transaction IDs that
	// contain TCP connect failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedTCPConnectFailures optional.Value[map[int64]bool]

	// TCPTransactionsWithUnexpectedTLSHandshakeFailures contains the TCP transaction IDs that
	// contain TLS handshake failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedTLSHandshakeFailures optional.Value[map[int64]bool]

	// TCPSTransactionsWithUnexpectedHTTPFailures contains the TCP transaction IDs that
	// contain HTTP failures while the control measurement succeeded.
	TCPTransactionsWithUnexpectedHTTPFailures optional.Value[map[int64]bool]
}

func analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" && obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
}

// ComputeDNSExperimentFailure computes the DNSExperimentFailure field.
func (wa *WebAnalysis) ComputeDNSExperimentFailure(c *WebObservationsContainer) {

	for _, obs := range c.DNSLookupFailures {
		// make sure we only include the system resolver
		switch obs.DNSEngine.UnwrapOr("") {
		case "getaddrinfo", "golang_net_resolver":

			// skip cases where there's no DNS record for AAAA, which is a false positive
			if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
				continue
			}

			// only record the first failure
			wa.DNSExperimentFailure = obs.DNSLookupFailure
			return

		default:
			// nothing
		}
	}
}

func analysisForEachDNSTransactionID(obs *WebObservation, f func(id int64)) {
	for _, id := range obs.DNSTransactionIDs.UnwrapOr(nil) {
		f(id)
	}
}

// ComputeDNSTransactionsWithBogons computes the DNSTransactionsWithBogons field.
func (wa *WebAnalysis) ComputeDNSTransactionsWithBogons(c *WebObservationsContainer) {
	// Implementation note: any bogon IP address resolved by a DoH service
	// is STILL suspicious since it should not happen. TODO(bassosimone): an
	// even better algorithm could possibly check whether also the TH has
	// observed bogon IP addrs and avoid flagging in such a case.
	//
	// See https://github.com/ooni/probe/issues/2274 for more information.

	state := make(map[int64]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// we're only interested in cases when there's a bogon
		if !obs.IPAddressBogon.UnwrapOr(false) {
			continue
		}

		// update state
		analysisForEachDNSTransactionID(obs, func(id int64) {
			state[id] = true
		})
	}

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

	state := make(map[int64]bool)

	for _, obs := range c.DNSLookupFailures {
		// skip cases where the control failed as well
		if obs.ControlDNSLookupFailure.UnwrapOr("") != "" {
			continue
		}

		// skip cases where the engine is doh (see above comment)
		if analysisDNSEngineIsDNSOverHTTPS(obs) {
			continue
		}

		// skip cases where there's no DNS record for AAAA, which is a false positive
		if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
			continue
		}

		// update state
		analysisForEachDNSTransactionID(obs, func(id int64) {
			state[id] = true
		})
	}

	wa.DNSTransactionsWithBogons = optional.Some(state)
}

// ComputeDNSPossiblyInvalidAddrs computes the DNSPossiblyInvalidAddrs field.
func (wa *WebAnalysis) ComputeDNSPossiblyInvalidAddrs(c *WebObservationsContainer) {
	// Implementation note: in the case in which DoH returned answers, here
	// it still feels okay to consider them. We should avoid flagging DoH
	// failures as measurement failures but if DoH returns us some unexpected
	// even-non-bogon addr, it seems worth flagging for now.
	//
	// See https://github.com/ooni/probe/issues/2274

	state := make(map[string]bool)

	for _, obs := range c.KnownTCPEndpoints {
		addr := obs.IPAddress.Unwrap()

		// if the address was also resolved by the control, we're good
		if obs.MatchWithControlIPAddress.UnwrapOr(false) {
			continue
		}

		// if we have a succesful TLS handshake for this addr, we're good
		if obs.TLSHandshakeFailure.UnwrapOr("unknown_failure") == "" {
			// just in case another transaction succeded, clear the address from the state
			delete(state, addr)
			continue
		}

		// if there's an ASN match with the control, we're good
		if obs.MatchWithControlIPAddressASN.UnwrapOr(false) {
			continue
		}

		// update state
		state[addr] = true
	}

	wa.DNSPossiblyInvalidAddrs = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexpectedTCPConnectFailures computes the TCPTransactionsWithUnexpectedTCPConnectFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedTCPConnectFailures(c *WebObservationsContainer) {
	state := make(map[int64]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.TCPConnectFailure.UnwrapOr("") == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTCPConnectFailure.UnwrapOr("unknown_failure") != "" {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	wa.TCPTransactionsWithUnexpectedTCPConnectFailures = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures computes the TCPTransactionsWithUnexpectedTLSHandshakeFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures(c *WebObservationsContainer) {
	state := make(map[int64]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.TLSHandshakeFailure.UnwrapOr("") == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlTLSHandshakeFailure.UnwrapOr("unknown_failure") != "" {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	wa.TCPTransactionsWithUnexpectedTLSHandshakeFailures = optional.Some(state)
}

// ComputeTCPTransactionsWithUnexpectedHTTPFailures computes the TCPTransactionsWithUnexpectedHTTPFailures field.
func (wa *WebAnalysis) ComputeTCPTransactionsWithUnexpectedHTTPFailures(c *WebObservationsContainer) {
	state := make(map[int64]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// skip cases with no failures
		if obs.HTTPFailure.UnwrapOr("") == "" {
			continue
		}

		// skip cases where also the control failed
		if obs.ControlHTTPFailure.UnwrapOr("unknown_failure") != "" {
			continue
		}

		// update state
		state[obs.EndpointTransactionID.Unwrap()] = true
	}

	wa.TCPTransactionsWithUnexpectedHTTPFailures = optional.Some(state)
}

// ComputeHTTPDiffBodyProportionFactor computes the HTTPDiffBodyProportionFactor field.
func (wa *WebAnalysis) ComputeHTTPDiffBodyProportionFactor(c *WebObservationsContainer) {
	for _, obs := range c.KnownTCPEndpoints {
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
		// we should only have control headers for the "final" response
		measurement := obs.HTTPResponseHeadersKeys.UnwrapOr(nil)
		if len(measurement) <= 0 {
			continue
		}
		control := obs.ControlHTTPResponseHeadersKeys.UnwrapOr(nil)
		if len(control) <= 0 {
			continue
		}

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
		break
	}

	wa.HTTPDiffUncommonHeadersIntersection = optional.Some(state)
}

// ComputeHTTPDiffTitleDifferentLongWords computes the HTTPDiffTitleDifferentLongWords field.
func (wa *WebAnalysis) ComputeHTTPDiffTitleDifferentLongWords(c *WebObservationsContainer) {
	state := make(map[string]bool)

	for _, obs := range c.KnownTCPEndpoints {
		// we should only have control headers for the "final" response
		measurement := obs.HTTPResponseTitle.UnwrapOr("")
		if measurement == "" {
			continue
		}
		control := obs.ControlHTTPResponseTitle.UnwrapOr("")
		if control == "" {
			continue
		}

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

		// check whether there's a long word that does not match
		for word, score := range words {
			if (score & (byProbe | byTH)) != (byProbe | byTH) {
				state[word] = true
				break
			}
		}

		// Implementation note: we only process the first observation that matches.
		//
		// This is fine(TM) as long as we have a single "final" request.
		break
	}

	wa.HTTPDiffTitleDifferentLongWords = optional.Some(state)
}
