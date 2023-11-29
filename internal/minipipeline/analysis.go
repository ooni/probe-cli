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
	// HTTPFinalResponseSuccessTLS contains the IDs of the final HTTP
	// responses for which we observed a successful TLS handshake.
	HTTPFinalResponseSuccessTLS Set[int64]

	// HTTPFinalResponseMissingControl contains the IDs of the final HTTP
	// responses for which we're missing control information.
	HTTPFinalResponseMissingControl Set[int64]

	// HTTPFinalResponseExpectedFailure contains the IDs of the final HTTP
	// responses for which both the probe and the control failed.
	HTTPFinalResponseExpectedFailure Set[int64]

	// HTTPFinalResponseUnexpectedSuccess contains the IDs of the final HTTP
	// responses for which the probe succeeded and the control failed.
	HTTPFinalResponseUnexpectedSuccess Set[int64]

	// HTTPFinalResponseUnexpectedFailure contains the IDs of the final HTTP
	// responses for which the probe failed and the control succeeded.
	HTTPFinalResponseUnexpectedFailure Set[int64]

	// HTTPDiffBodyProportionFactor contains the body proportion factor for the first
	// "final" HTTP request inside the dataset compared to the control.
	HTTPDiffBodyProportionFactor optional.Value[float64]

	// HTTPDiffStatusCodeMatch returns whether the status code matches for the first
	// "final" HTTP request inside the dataset compared to the control.
	HTTPDiffStatusCodeMatch optional.Value[bool]

	// HTTPDiffUncommonHeadersIntersections contains the uncommon headers intersection for the first
	// "final" HTTP request inside the dataset compared to the control.
	HTTPDiffUncommonHeadersIntersection optional.Value[Set[string]]

	// HTTPDiffTitleDifferentLongWords contains the words long 5+ characters that appear
	// "final" HTTP request inside the dataset compared to the control.
	HTTPDiffTitleDifferentLongWords optional.Value[Set[string]]

	// TLSHandshakeSuccessWithoutControl contains the IDs of the endpoint transactions
	// where the TLS handshake succeded without control information.
	TLSHandshakeSuccessWithoutControl Set[int64]

	// TLSHandshakeFailureWithoutControl contains the IDs of the endpoint transactions
	// where the TLS handshake failed without control information.
	TLSHandshakeFailureWithoutControl Set[int64]

	// TLSHandshakeExpectedFailure contains the IDs of the endpoint transactions
	// where the TLS handshake failed for the probe and the control.
	TLSHandshakeExpectedFailure Set[int64]

	// TLSHandshakeUnexpectedSuccess contains the IDs of the endpoint transactions
	// where the probe succeeded and the control failed.
	TLSHandshakeUnexpectedSuccess Set[int64]

	// TLSHandshakeUnexpectedFailure contains the IDs of the endpoint transactions
	// where the probe failed and the control succeeded.
	TLSHandshakeUnexpectedFailure Set[int64]

	// TLSHandshakeExpectedSuccess contains the IDs of the endpoint transactions
	// where the probe succeeded and the control succeeded.
	TLSHandshakeExpectedSuccess Set[int64]

	// TCPConnectFailureWithoutControl contains the IDs of the endpoint transactions
	// where the probe failed without control info.
	TCPConnectFailureWithoutControl Set[int64]

	// TCPConnectSuccessWithoutControl contains the IDs of the endpoint transactions
	// where the probe succeeded without control info.
	TCPConnectSuccessWithoutControl Set[int64]

	// TCPConnectExpectedFailure contains the IDs of the endpoint transactions
	// where the probe failed and also the control failed.
	TCPConnectExpectedFailure Set[int64]

	// TCPConnectUnexpectedSuccess contains the IDs of the endpoint transactions
	// where the probe succeeded and the control failed.
	TCPConnectUnexpectedSuccess Set[int64]

	// TCPConnectUnexpectedFailure contains the IDs of the endpoint transactions
	// where the probe failed and the control succeeded.
	TCPConnectUnexpectedFailure Set[int64]

	// TCPConnectExpectedSuccess contains the IDs of the endpoint transactions
	// where the probe succeeded and the control succeeded.
	TCPConnectExpectedSuccess Set[int64]

	// IPAddressesValidatedUsingTLS contains the IP addresses that we validated
	// by performing a successful TLS handshake.
	IPAddressesValidatedUsingTLS Set[string]

	// IPAddressesValidatedByEquality contains all the IP addresses that
	// were resolved both by the probe and the control.
	IPAddressesValidatedByEquality Set[string]

	// IPAddressesValidatedByASN contains all the IP addresses whose ASNs
	// were resolved both by the probe and the control.
	IPAddressesValidatedByASN Set[string]

	// IPAddressesBogons contains all the IP addresses that were bogons.
	IPAddressesBogons Set[string]

	// IPAddressesAll contains all the observed IP addresses.
	IPAddressesAll Set[string]

	// DNSLookupFailureWithoutControl contains the IDs of the DNS transactions
	// where the probe failed and there's no control.
	DNSLookupFailureWithoutControl Set[int64]

	// DNSLookupSuccessWithoutControl contains the IDs of the DNS transactions
	// where the probe succeeded and there's no control.
	DNSLookupSuccessWithoutControl Set[int64]

	// DNSLookupExpectedFailure contains the IDs of the DNS transactions
	// where the probe and the control failed.
	DNSLookupExpectedFailure Set[int64]

	// DNSLookupUnexpectedSuccess contains the IDs of the DNS transactions
	// where the probe succeeded and the control failed.
	DNSLookupUnexpectedSuccess Set[int64]

	// DNSLookupUnexpectedFailure contains the IDs of the DNS transactions
	// where the probe failed and the control succeeded.
	DNSLookupUnexpectedFailure Set[int64]

	// DNSLookupExpectedSuccess contains the IDs of the DNS transactions
	// where the probe succeeded and the control succeeded.
	DNSLookupExpectedSuccess Set[int64]

	// DNSExperimentFailure is the first DNS failure in the dataset.
	DNSExperimentFailure optional.Value[string]

	// HTTPExperimentFailure is the first HTTP failure in the dataset.
	HTTPExperimentFailure optional.Value[string]
}

func analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs *WebObservation) bool {
	return obs.DNSQueryType.UnwrapOr("") == "AAAA" &&
		obs.DNSLookupFailure.UnwrapOr("") == netxlite.FailureDNSNoAnswer
}

func (w *WebAnalysis) analyzeWebObservationsContainer(c *WebObservationsContainer) {
	for _, obs := range c.DNSLookupFailures {
		w.analyzeDNSLookup(obs)
	}
	for _, obs := range c.KnownTCPEndpoints {
		w.analyzeTCPEndpoint(obs)
	}
}

func (w *WebAnalysis) analyzeTCPEndpoint(obs *WebObservation) {
	// HTTPFinalResponse
	w.analyzeHTTPFinalResponse(obs)

	// TLSHandshake
	w.analyzeTLSHandshake(obs)

	// TCPConnect
	w.analyzeTCPConnect(obs)

	// IPAddress
	w.analyzeIPAddress(obs)
}

func (w *WebAnalysis) analyzeHTTPFinalResponse(obs *WebObservation) {
	// we need a final HTTP response
	if obs.HTTPResponseIsFinal.IsNone() || !obs.HTTPResponseIsFinal.Unwrap() {
		return
	}

	// set the HTTPExperimentFailure
	if w.HTTPExperimentFailure.IsNone() && obs.HTTPFailure.Unwrap() != "" {
		w.HTTPExperimentFailure = obs.HTTPFailure
	}

	// handle the case where we TLS succeed
	if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
		w.HTTPFinalResponseSuccessTLS.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where there's no control
	if obs.ControlHTTPFailure.IsNone() {
		w.HTTPFinalResponseMissingControl.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where probe and control both fail
	if obs.HTTPFailure.Unwrap() != "" && obs.ControlHTTPFailure.Unwrap() != "" {
		w.HTTPFinalResponseExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the control fails
	if obs.ControlHTTPFailure.Unwrap() != "" {
		w.HTTPFinalResponseUnexpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe fails
	if obs.HTTPFailure.Unwrap() != "" {
		w.HTTPFinalResponseUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// check for HTTP diff
	w.analyzeHTTPDiffBodyProportionFactor(obs)
	w.analyzeHTTPDiffStatusCodeMatch(obs)
	w.analyzeHTTPDiffUncommonHeadersIntersection(obs)
	w.analyzeHTTPDiffTitleDifferentLongWords(obs)
}

func analysisDNSEngineIsDNSOverHTTPS(obs *WebObservation) bool {
	return obs.DNSEngine.UnwrapOr("") == "doh"
}

func (w *WebAnalysis) analyzeDNSLookup(obs *WebObservation) {
	// Implementation note: a DoH failure is not information about the URL we're
	// measuring but about the DoH service being blocked.
	//
	// See https://github.com/ooni/probe/issues/2274
	if analysisDNSEngineIsDNSOverHTTPS(obs) {
		return
	}

	// skip cases where there's no DNS record for AAAA, which is a false positive
	if analysisDNSLookupFailureIsDNSNoAnswerForAAAA(obs) {
		return
	}

	// TODO(bassosimone): if we set an IPv6 address as the resolver address, we
	// end up with false positive errors when there's no IPv6 support

	// set the DNSExperimentFailure
	if w.DNSExperimentFailure.IsNone() && obs.DNSLookupFailure.Unwrap() != "" {
		w.DNSExperimentFailure = obs.DNSLookupFailure
	}

	// handle the case where there's no control
	if obs.ControlDNSLookupFailure.IsNone() {
		if obs.DNSLookupFailure.Unwrap() != "" {
			w.DNSLookupFailureWithoutControl.Add(obs.DNSTransactionID.Unwrap())
			return
		}
		w.DNSLookupSuccessWithoutControl.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where both failed
	if obs.DNSLookupFailure.Unwrap() != "" && obs.ControlDNSLookupFailure.Unwrap() != "" {
		w.DNSLookupExpectedFailure.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where only the control failed
	if obs.ControlDNSLookupFailure.Unwrap() != "" {
		w.DNSLookupUnexpectedSuccess.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe failed
	if obs.DNSLookupFailure.Unwrap() != "" {
		w.DNSLookupUnexpectedFailure.Add(obs.DNSTransactionID.Unwrap())
		return
	}

	// handle the case where both succeeded
	w.DNSLookupExpectedSuccess.Add(obs.DNSTransactionID.Unwrap())
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

func (w *WebAnalysis) analyzeIPAddress(obs *WebObservation) {
	if !obs.MatchWithControlIPAddress.IsNone() && obs.MatchWithControlIPAddress.Unwrap() {
		w.IPAddressesValidatedByEquality.Add(obs.IPAddress.Unwrap())
	}

	if !obs.MatchWithControlIPAddressASN.IsNone() && obs.MatchWithControlIPAddressASN.Unwrap() {
		w.IPAddressesValidatedByASN.Add(obs.IPAddress.Unwrap())
	}

	w.IPAddressesAll.Add(obs.IPAddress.Unwrap())
}

func (w *WebAnalysis) analyzeTLSHandshake(obs *WebObservation) {
	// we need a valid TLS handshake
	if obs.TLSHandshakeFailure.IsNone() {
		return
	}

	// handle the case where there is no control information
	if obs.ControlTLSHandshakeFailure.IsNone() {
		if obs.TLSHandshakeFailure.Unwrap() != "" {
			w.TLSHandshakeFailureWithoutControl.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.TLSHandshakeSuccessWithoutControl.Add(obs.EndpointTransactionID.Unwrap())
		w.IPAddressesValidatedUsingTLS.Add(obs.IPAddress.Unwrap())
		return
	}

	// handle the case where both the probe and the control fail
	if obs.TLSHandshakeFailure.Unwrap() != "" && obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the control fails
	if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeUnexpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
		w.IPAddressesValidatedUsingTLS.Add(obs.IPAddress.Unwrap())
		return
	}

	// handle the case where only the probe fails
	if obs.TLSHandshakeFailure.Unwrap() != "" {
		w.TLSHandshakeUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both succeed
	w.TLSHandshakeExpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
	w.IPAddressesValidatedUsingTLS.Add(obs.IPAddress.Unwrap())
}

func (w *WebAnalysis) analyzeTCPConnect(obs *WebObservation) {
	if obs.TCPConnectFailure.IsNone() {
		return
	}

	// handle the case where there is no control information
	if obs.ControlTCPConnectFailure.IsNone() {
		if obs.TCPConnectFailure.Unwrap() != "" {
			w.TCPConnectFailureWithoutControl.Add(obs.EndpointTransactionID.Unwrap())
			return
		}
		w.TCPConnectSuccessWithoutControl.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both the probe and the control fail
	if obs.TCPConnectFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
		w.TCPConnectExpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the control fails
	if obs.ControlTCPConnectFailure.Unwrap() != "" {
		w.TCPConnectUnexpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where only the probe fails
	if obs.TCPConnectFailure.Unwrap() != "" {
		if analysisTCPConnectFailureSeemsMisconfiguredIPv6(obs) {
			return
		}
		w.TCPConnectUnexpectedFailure.Add(obs.EndpointTransactionID.Unwrap())
		return
	}

	// handle the case where both succeed
	w.TCPConnectExpectedSuccess.Add(obs.EndpointTransactionID.Unwrap())
}

func (w *WebAnalysis) analyzeHTTPDiffBodyProportionFactor(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPDiffBodyProportionFactor.IsNone() {
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
	w.HTTPDiffBodyProportionFactor = optional.Some(proportion)
}

func (w *WebAnalysis) analyzeHTTPDiffStatusCodeMatch(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPDiffStatusCodeMatch.IsNone() {
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
	w.HTTPDiffStatusCodeMatch = optional.Some(good)
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
	if !w.HTTPDiffUncommonHeadersIntersection.IsNone() {
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
	w.HTTPDiffUncommonHeadersIntersection = optional.Some(state)
}

func (w *WebAnalysis) analyzeHTTPDiffTitleDifferentLongWords(obs *WebObservation) {
	// skip if we have already run
	if !w.HTTPDiffTitleDifferentLongWords.IsNone() {
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

	w.HTTPDiffTitleDifferentLongWords = optional.Some(state)
}
