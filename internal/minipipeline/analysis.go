package minipipeline

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// AnalyzeWebObservations generates a [*WebAnalysis] from a [*WebObservationsContainer].
func AnalyzeWebObservations(container *WebObservationsContainer) *WebAnalysis {
	analysis := &WebAnalysis{}
	analysis.analyzeWebObservationsContainer(container)
	return analysis
}

// WebAnalysis summarizes the content of [*WebObservationsContainer].
//
// The zero value of this struct is ready to use.
type WebAnalysis struct {
	// These fields classify the "final" HTTP responses. A final response did not
	// fail at the HTTP layer and has a 2xx, 4xx, or 5xx status code.
	//
	// We classify all the responses depending on whether:
	//
	// 1. control information is available;
	//
	// 2. they used TLS or TCP.
	//
	// When control information is available, we compare the final response
	// status code, HTTP headers, body, and title with the control.
	HTTPFinalResponseWithoutControlTLS Set[int64]
	HTTPFinalResponseWithoutControlTCP Set[int64]
	HTTPFinalResponseWithControlTLS    Set[int64]
	HTTPFinalResponseWithControlTCP    Set[int64]

	// These fields show the comparison between the first final response
	// with control and the corresponding control response.
	HTTPFinalResponseDiffBodyProportionFactor        optional.Value[float64]
	HTTPFinalResponseDiffStatusCodeMatch             optional.Value[bool]
	HTTPFinalResponseDiffUncommonHeadersIntersection optional.Value[Set[string]]
	HTTPFinalResponseDiffTitleDifferentLongWords     optional.Value[Set[string]]

	// These fields classify the "non-final" HTTP responses. Since they're not
	// final, they either fail or succeed with a 3xx status code.
	HTTPNonFinalResponseSuccessTLS                   Set[int64]
	HTTPNonFinalResponseSuccessTCP                   Set[int64]
	HTTPNonFinalResponseFailureWithoutControl        Set[int64]
	HTTPNonFinalResponseFailureWithControlExpected   Set[int64]
	HTTPNonFinalResponseFailureWithControlUnexpected Set[int64]

	// These fields classify TLS handshakes.
	TLSHandshakeWithoutControlFailure        Set[int64]
	TLSHandshakeWithoutControlSuccess        Set[int64]
	TLSHandshakeWithControlExpectedFailure   Set[int64]
	TLSHandshakeWithControlUnexpectedSuccess Set[int64]
	TLSHandshakeWithControlUnexpectedFailure Set[int64]
	TLSHandshakeWithControlExpectedSuccess   Set[int64]

	// These fields classify TCP connects.
	TCPConnectWithoutControlFailure                 Set[int64]
	TCPConnectWithoutControlSuccess                 Set[int64]
	TCPConnectWithControlExpectedFailure            Set[int64]
	TCPConnectWithControlUnexpectedSuccess          Set[int64]
	TCPConnectWithControlUnexpectedFailure          Set[int64]
	TCPConnectWithControlUnexpectedFailureMaybeIPv6 Set[int64]
	TCPConnectWithControlUnexpectedFailureAnomaly   Set[int64]
	TCPConnectWithControlExpectedSuccess            Set[int64]

	// These fields classify IP addresses used by endpoints.
	EndpointIPAddressesValidTLS               Set[string]
	EndpointIPAddressesInvalidBogon           Set[string]
	EndpointIPAddressesUnknown                Set[string]
	EndpointIPAddressesControlValidByEquality Set[string]
	EndpointIPAddressesControlValidByASN      Set[string]
	EndpointIPAddressesControlInvalid         Set[string]

	// These fields classify DNS lookups.
	DNSLookupHTTPS                        Set[int64]
	DNSLookupAAAANoAnswer                 Set[int64]
	DNSLookupWithoutControlFailure        Set[int64]
	DNSLookupWithoutControlSuccess        Set[int64]
	DNSLookupWithControlExpectedFailure   Set[int64]
	DNSLookupWithControlUnexpectedSuccess Set[int64]
	DNSLookupWithControlUnexpectedFailure Set[int64]
	DNSLookupWithControlExpectedSuccess   Set[int64]
}

func (w *WebAnalysis) analyzeWebObservationsContainer(c *WebObservationsContainer) {
	for _, obs := range c.DNSLookupFailures {
		w.analyzeDNSLookup(obs)
	}
	for _, obs := range c.KnownTCPEndpoints {
		w.analyzeTCPEndpoint(obs)
	}
}

func (w *WebAnalysis) analyzeDNSLookup(obs *WebObservation) {
	// we must have a DNS lookup failure
	if obs.DNSLookupFailure.IsNone() {
		return
	}

	// Implementation note: a DoH failure is not information about the URL we're
	// measuring but about the DoH service being blocked.
	//
	// See https://github.com/ooni/probe/issues/2274
	if analysisDNSEngineIsDNSOverHTTPS(obs) {
		w.DNSLookupHTTPS.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// skip cases where there's no DNS record for AAAA, which is a false positive
	if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
		w.DNSLookupAAAANoAnswer.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// TODO(bassosimone): if we set an IPv6 address as the resolver address, we
	// end up with false positive errors when there's no IPv6 support

	// handle the case where there's no control
	if obs.ControlDNSLookupFailure.IsNone() {
		if obs.DNSLookupFailure.Unwrap() != "" {
			w.DNSLookupWithoutControlFailure.Add(obs.DNSTransactionID.Unwrap())
			return
		}
		w.DNSLookupWithoutControlSuccess.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where both failed
	if obs.DNSLookupFailure.Unwrap() != "" && obs.ControlDNSLookupFailure.Unwrap() != "" {
		w.DNSLookupWithControlExpectedFailure.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where only the control failed
	if obs.ControlDNSLookupFailure.Unwrap() != "" {
		w.DNSLookupWithControlUnexpectedSuccess.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe failed
	if obs.DNSLookupFailure.Unwrap() != "" {
		w.DNSLookupWithControlUnexpectedFailure.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where both succeeded
	w.DNSLookupWithControlExpectedSuccess.Add(obs.DNSTransactionID.Unwrap())
}

func (w *WebAnalysis) analyzeTCPEndpoint(obs *WebObservation) {
	w.analyzeHTTPRoundTrip(obs)
	w.analyzeTLSHandshake(obs)
	w.analyzeTCPConnect(obs)
	w.analyzeEndpointIPAddress(obs)
}

func (w *WebAnalysis) analyzeHTTPRoundTrip(obs *WebObservation) {
	// we need a final HTTP response
	if obs.HTTPResponseIsFinal.IsNone() || !obs.HTTPResponseIsFinal.Unwrap() {
		// there needs to be a defined failure
		if obs.HTTPFailure.IsNone() {
			return
		}

		// handle and classify the case of failure
		//
		// when there is a control, we know the expectation for the final
		// response, so we can determine whether there's blocking
		if obs.HTTPFailure.Unwrap() != "" {
			if obs.ControlHTTPFailure.IsNone() {
				w.HTTPNonFinalResponseFailureWithoutControl.Add(obs.EndpointTransactionID.Unwrap())
				return
			}
			if obs.ControlHTTPFailure.Unwrap() != "" {
				w.HTTPNonFinalResponseFailureWithControlExpected.Add(obs.EndpointTransactionID.Unwrap())
				return
			}
			w.HTTPNonFinalResponseFailureWithControlUnexpected.Add(obs.EndpointTransactionID.Unwrap())
			return
		}

		// handle and classify the case of success
		if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
			w.HTTPNonFinalResponseSuccessTLS.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.HTTPNonFinalResponseSuccessTCP.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where there's no control
	if obs.ControlHTTPFailure.IsNone() {
		if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
			w.HTTPFinalResponseWithoutControlTLS.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.HTTPFinalResponseWithoutControlTCP.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// count and classify the number of final responses with control
	if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
		w.HTTPFinalResponseWithControlTLS.Add(obs.EndpointTransactionID.Unwrap())
	} else {
		w.HTTPFinalResponseWithControlTCP.Add(obs.EndpointTransactionID.Unwrap())
	}

	// compute the HTTPDiff metrics
	w.analyzeHTTPDiffBodyProportionFactor(obs)
	w.analyzeHTTPDiffStatusCodeMatch(obs)
	w.analyzeHTTPDiffUncommonHeadersIntersection(obs)
	w.analyzeHTTPDiffTitleDifferentLongWords(obs)
}

func (w *WebAnalysis) analyzeTLSHandshake(obs *WebObservation) {
	// we need a valid TLS handshake
	if obs.TLSHandshakeFailure.IsNone() {
		return
	}

	// handle the case where there is no control information
	if obs.ControlTLSHandshakeFailure.IsNone() {
		if obs.TLSHandshakeFailure.Unwrap() != "" {
			w.TLSHandshakeWithoutControlFailure.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.TLSHandshakeWithoutControlSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both the probe and the control fail
	if obs.TLSHandshakeFailure.Unwrap() != "" && obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeWithControlExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the control fails
	if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeWithControlUnexpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe fails
	if obs.TLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeWithControlUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both succeed
	w.TLSHandshakeWithControlExpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
}

func (w *WebAnalysis) analyzeTCPConnect(obs *WebObservation) {
	// we need a valid TCP connect attempt
	if obs.TCPConnectFailure.IsNone() {
		return
	}

	// handle the case where there is no control information
	if obs.ControlTCPConnectFailure.IsNone() {
		if obs.TCPConnectFailure.Unwrap() != "" {
			w.TCPConnectWithoutControlFailure.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.TCPConnectWithoutControlSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both the probe and the control fail
	if obs.TCPConnectFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
		w.TCPConnectWithControlExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the control fails
	if obs.ControlTCPConnectFailure.Unwrap() != "" {
		w.TCPConnectWithControlUnexpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe fails
	if obs.TCPConnectFailure.Unwrap() != "" {
		w.TCPConnectWithControlUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		if analysisTCPConnectFailureSeemsMisconfiguredIPv6(obs) {
			w.TCPConnectWithControlUnexpectedFailureMaybeIPv6.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.TCPConnectWithControlUnexpectedFailureAnomaly.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both succeed
	w.TCPConnectWithControlExpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
}

func (w *WebAnalysis) analyzeEndpointIPAddress(obs *WebObservation) {
	// check whether it's invalid because it's a bogon
	if !obs.IPAddressBogon.IsNone() && obs.IPAddressBogon.Unwrap() {
		w.EndpointIPAddressesInvalidBogon.Add(obs.IPAddress.Unwrap())
		return
	}

	// check whether it's valid because of TLS
	if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
		w.EndpointIPAddressesValidTLS.Add(obs.IPAddress.Unwrap())
		return
	}

	// if we don't know the control failure, this endpoint was not matched
	// with a control, so say that we really don't know
	if obs.ControlDNSLookupFailure.IsNone() {
		w.EndpointIPAddressesUnknown.Add(obs.IPAddress.Unwrap())
		return
	}

	// check whether it's valid by equality with the control
	if !obs.MatchWithControlIPAddress.IsNone() && obs.MatchWithControlIPAddress.Unwrap() {
		w.EndpointIPAddressesControlValidByEquality.Add(obs.IPAddress.Unwrap())
		return
	}

	// check whether it's valid because the control resolved the same set of ASNs
	if !obs.MatchWithControlIPAddressASN.IsNone() && obs.MatchWithControlIPAddressASN.Unwrap() {
		w.EndpointIPAddressesControlValidByASN.Add(obs.IPAddress.Unwrap())
		return
	}

	// otherwise the control says this IP address is not valid
	w.EndpointIPAddressesControlInvalid.Add(obs.IPAddress.Unwrap())
}

func (w *WebAnalysis) analyzeHTTPDiffBodyProportionFactor(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPFinalResponseDiffBodyProportionFactor.IsNone() {
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

	// compute the body proportion factor
	var proportion float64
	if measurement >= control {
		proportion = float64(control) / float64(measurement)
	} else {
		proportion = float64(measurement) / float64(control)
	}

	// update state
	w.HTTPFinalResponseDiffBodyProportionFactor = optional.Some(proportion)
}

func (w *WebAnalysis) analyzeHTTPDiffStatusCodeMatch(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPFinalResponseDiffStatusCodeMatch.IsNone() {
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
	w.HTTPFinalResponseDiffStatusCodeMatch = optional.Some(good)
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

func (w *WebAnalysis) analyzeHTTPDiffUncommonHeadersIntersection(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPFinalResponseDiffUncommonHeadersIntersection.IsNone() {
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
	var state Set[string]
	for key, value := range matching {
		if (value & (byProbe | byTH)) == (byProbe | byTH) {
			state.Add(key)
		}
	}
	w.HTTPFinalResponseDiffUncommonHeadersIntersection = optional.Some(state)
}

func (w *WebAnalysis) analyzeHTTPDiffTitleDifferentLongWords(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPFinalResponseDiffTitleDifferentLongWords.IsNone() {
		return
	}

	// We should only perform the comparison if we have valid control data. Because
	// the title could legitimately be empty, let's use the status code here.
	if obs.ControlHTTPResponseStatusCode.UnwrapOr(0) <= 0 {
		return
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
	var state Set[string]
	for word, score := range words {
		if (score & (byProbe | byTH)) != (byProbe | byTH) {
			state.Add(word)
		}
	}

	w.HTTPFinalResponseDiffTitleDifferentLongWords = optional.Some(state)
}

func analysisDNSEngineIsDNSOverHTTPS(obs *WebObservation) bool {
	return obs.DNSEngine.UnwrapOr("") == "doh"
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

func analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" &&
		obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
}
