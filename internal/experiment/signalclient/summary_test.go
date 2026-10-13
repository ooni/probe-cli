package signalclient

import "testing"

func TestMeasurementSummaryKeys(t *testing.T) {
	failure := func(s string) *string { return &s }
	ok := &TargetResult{Name: "chat_libsignal"}
	for _, c := range []struct {
		name    string
		status  string
		targets []*TargetResult
		want    bool
	}{
		{"all reached", "ok", []*TargetResult{ok, ok}, false},
		{"TLS blocked", "ok", []*TargetResult{ok, {Failure: failure("connection_reset")}}, true},
		{"DNS blocked", "ok", []*TargetResult{{Failure: failure("dns_nxdomain_error")}}, true},
		{"stalled after the handshake", "ok", []*TargetResult{{Failure: failure("generic_timeout_error")}}, true},
		{"refused by the server", "ok", []*TargetResult{ok, {Failure: failure(FailureWebSocketUpgradeRefused)}}, false},
		{"malformed message from the server", "ok", []*TargetResult{{Failure: failure(FailureMalformedMessage)}}, false},
		{"the beacon reports an outage (it may be forged)", "outage", []*TargetResult{{Failure: failure("connection_reset")}}, true},
		{"the beacon reports an outage, targets reached", "outage", []*TargetResult{ok}, false},
		{"beacon lookup failed too", "dns_nxdomain_error", []*TargetResult{{Failure: failure("dns_nxdomain_error")}}, true},
		{"no targets", "ok", nil, false},
	} {
		tk := &TestKeys{SignalServiceStatus: c.status, Targets: c.targets}
		if got := tk.MeasurementSummaryKeys().Anomaly(); got != c.want {
			t.Errorf("%s: anomaly %v, want %v", c.name, got, c.want)
		}
	}
}
