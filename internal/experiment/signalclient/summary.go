package signalclient

//
// The probe-side verdict (ts-041 Probe-side verdict): what ooniprobe and the apps show the user.
// Conclusions about how Signal is blocked are left to the backend; this only says whether a
// direct target failed the way a censor makes it fail.
//

import "github.com/ooni/probe-cli/v3/internal/model"

// answeredFailures are failures that only the server can cause: every target's TLS is verified
// against Signal's roots, so these come from Signal or the CDN in front of it (for example a
// refusal, which can be a geoblock or rate limit), never from a middlebox, which can only reset,
// drop, stall or close. They are not an anomaly by themselves.
var answeredFailures = map[string]bool{
	FailureWebSocketUpgradeRefused:     true,
	FailureWebSocketUpgradeInvalid:     true,
	FailureExtendedConnectNotSupported: true,
	FailureUnexpectedMessage:           true,
	FailureMalformedMessage:            true,
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

var _ model.MeasurementSummaryKeysProvider = &TestKeys{}

// MeasurementSummaryKeys implements model.MeasurementSummaryKeysProvider: an anomaly is a direct
// target that failed at the network level (DNS, TCP, TLS, or a reset, EOF or timeout later on).
// Controls and circumvention routes do not count. Nor does signal_service_status: the beacon is
// resolved through the network being measured, so a censor could forge an outage to hide the
// blocking; outages are judged by the backend across networks (ts-029 ignores it too).
func (tk *TestKeys) MeasurementSummaryKeys() model.MeasurementSummaryKeys {
	sk := &SummaryKeys{}
	for _, r := range tk.Targets {
		if r != nil && r.Failure != nil && !answeredFailures[*r.Failure] {
			sk.IsAnomaly = true
		}
	}
	return sk
}

// Anomaly implements model.MeasurementSummaryKeys.
func (sk *SummaryKeys) Anomaly() bool {
	return sk.IsAnomaly
}
