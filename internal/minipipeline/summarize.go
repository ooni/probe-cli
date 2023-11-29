package minipipeline

// These constants define the index of bits in the [*Bitmask] returned by [*Summarize].
const (
	SummaryHTTPFinalResponseSuccessTLS = iota
	SummaryHTTPFinalResponseMissingControl
	SummaryHTTPFinalResponseWebsiteDown
	SummaryHTTPFinalResponseUnexpectedSuccess
	SummaryHTTPFinalResponseUnexpectedError
	SummaryHTTPFinalResponseSuccessTCP

	SummaryHTTPFinalResponseStatusCodeMatch
	SummaryHTTPFinalResponseStatusCodeMismatch
	// TODO...

	SummaryTLSHandshakeValidAddress
	SummaryTLSHandshakeMissingControl
	SummaryTLSHandshakeWebsiteDown
	SummaryTLSHandshakeUnexpectedSuccess
	SummaryTLSHandshakeUnexpectedError

	SummaryTCPConnectMissingControl
	SummaryTCPConnectWebsiteDown
	SummaryTCPConnectUnexpectedSuccess
	SummaryTCPConnectUnexpectedError
	SummaryTCPConnectValidAddress

	SummaryAddressBogon
	SummaryAddressResolvedByProbeAndTH
	SummaryAddressASNSeemsGood
)

// Summarize creates a bitmask summary of a [*WebObservation].
func Summarize(obs *WebObservation) *Bitmask {
	bitmask := &Bitmask{}

	if !obs.HTTPResponseIsFinal.IsNone() && obs.HTTPResponseIsFinal.Unwrap() {
		httpFinalResponseSummarizeFailure(obs, bitmask)
		httpFinalResponseSummarizeStatusCode(obs, bitmask)
		// TODO...
	}

	addressSummarize(obs, bitmask)
	tlsHandshakeSummarize(obs, bitmask)
	tcpConnectSummarize(obs, bitmask)

	return bitmask
}

func httpFinalResponseSummarizeFailure(obs *WebObservation, bitmask *Bitmask) {
	// SummaryHTTPFinalResponseSuccessTLS
	if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
		bitmask.Set(SummaryHTTPFinalResponseSuccessTLS)
		return
	}

	// SummaryHTTPFinalResponseMissingControl
	if obs.ControlHTTPFailure.IsNone() {
		bitmask.Set(SummaryHTTPFinalResponseMissingControl)
		return
	}

	// SummaryHTTPFinalResponseWebsiteDown
	if obs.HTTPFailure.Unwrap() != "" && obs.ControlHTTPFailure.Unwrap() != "" {
		bitmask.Set(SummaryHTTPFinalResponseWebsiteDown)
		return
	}

	// SummaryHTTPFinalResponseUnexpectedSuccess
	if obs.ControlHTTPFailure.Unwrap() != "" {
		bitmask.Set(SummaryHTTPFinalResponseUnexpectedSuccess)
		return
	}

	// SummaryHTTPFinalResponseUnexpectedError
	if obs.HTTPFailure.Unwrap() != "" {
		bitmask.Set(SummaryHTTPFinalResponseUnexpectedError)
		return
	}

	// SummaryHTTPFinalResponseSuccessTCP
	bitmask.Set(SummaryHTTPFinalResponseSuccessTCP)
}

func httpFinalResponseSummarizeStatusCode(obs *WebObservation, bitmask *Bitmask) {
	if obs.HTTPResponseStatusCode.IsNone() || obs.ControlHTTPResponseStatusCode.IsNone() {
		return
	}

	// compute whether there's a match including caveats
	measurement := obs.HTTPResponseStatusCode.Unwrap()
	control := obs.ControlHTTPResponseStatusCode.Unwrap()
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
	switch good {
	case true:
		bitmask.Set(SummaryHTTPFinalResponseStatusCodeMatch)
	case false:
		bitmask.Set(SummaryHTTPFinalResponseStatusCodeMismatch)
	}
}

func tlsHandshakeSummarize(obs *WebObservation, bitmask *Bitmask) {
	// SummaryTLSHandshakeValidAddress
	if !obs.TLSHandshakeFailure.IsNone() && obs.TLSHandshakeFailure.Unwrap() == "" {
		bitmask.Set(SummaryTLSHandshakeValidAddress)
		return
	}

	// SummaryTLSHandshakeMissingControl
	if obs.ControlTLSHandshakeFailure.IsNone() {
		bitmask.Set(SummaryTLSHandshakeMissingControl)
		return
	}

	// SummaryTLSHandshakeWebsiteDown
	if obs.TLSHandshakeFailure.Unwrap() != "" && obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		bitmask.Set(SummaryTLSHandshakeWebsiteDown)
		return
	}

	// SummaryTLSHandshakeUnexpectedSuccess
	if obs.ControlTLSHandshakeFailure.Unwrap() != "" {
		bitmask.Set(SummaryTLSHandshakeUnexpectedSuccess)
		return
	}

	// SummaryTLSHandshakeUnexpectedError
	if obs.TLSHandshakeFailure.Unwrap() != "" {
		bitmask.Set(SummaryTLSHandshakeUnexpectedError)
		return
	}
}

func tcpConnectSummarize(obs *WebObservation, bitmask *Bitmask) {
	// SummaryTCPConnectMissingControl
	if obs.ControlTCPConnectFailure.IsNone() {
		bitmask.Set(SummaryTCPConnectMissingControl)
		return
	}

	// SummaryTCPConnectWebsiteDown
	if obs.TCPConnectFailure.Unwrap() != "" && obs.ControlTCPConnectFailure.Unwrap() != "" {
		bitmask.Set(SummaryTCPConnectWebsiteDown)
		return
	}

	// SummaryTCPConnectUnexpectedSuccess
	if obs.ControlTCPConnectFailure.Unwrap() != "" {
		bitmask.Set(SummaryTCPConnectUnexpectedSuccess)
		return
	}

	// SummaryTCPConnectUnexpectedError
	if obs.TCPConnectFailure.Unwrap() != "" {
		bitmask.Set(SummaryTCPConnectUnexpectedError)
		return
	}

	// SummaryTCPConnectValidAddress
	bitmask.Set(SummaryTCPConnectValidAddress)
}

func addressSummarize(obs *WebObservation, bitmask *Bitmask) {
	// SummaryAddressBogon
	if !obs.IPAddressBogon.IsNone() && obs.IPAddressBogon.Unwrap() {
		bitmask.Set(SummaryAddressBogon)
	}

	// SummaryAddressResolvedByProbeAndTH
	if !obs.MatchWithControlIPAddress.IsNone() && obs.MatchWithControlIPAddress.Unwrap() {
		bitmask.Set(SummaryAddressResolvedByProbeAndTH)
	}

	// SummaryAddressASNSeemsGood
	if !obs.MatchWithControlIPAddressASN.IsNone() && obs.MatchWithControlIPAddressASN.Unwrap() {
		bitmask.Set(SummaryAddressASNSeemsGood)
	}
}
