package signalclient

//
// The data format (ts-041 Data format): the test keys, and how the per-target, per-route and
// per-control records are merged and finalized.
//

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TargetResult is the outcome for one target.
type TargetResult struct {
	Name            string              `json:"name"`
	Hostname        string              `json:"hostname"`
	Address         *string             `json:"address"` // null inside a reflector tunnel
	ClientHello     string              `json:"client_hello"`
	Method          Method              `json:"method"`
	UserAgent       string              `json:"user_agent"`
	ALPN            string              `json:"alpn"`
	LeafKeyType     string              `json:"leaf_key_type"`
	Status          int64               `json:"status"`
	SignalTimestamp bool                `json:"x_signal_timestamp"`
	WSExtensions    string              `json:"ws_extensions,omitempty"`
	Provisioning    *ProvisioningResult `json:"provisioning,omitempty"`
	Response        *ResponseSample     `json:"response,omitempty"` // https_get, or a refused upgrade
	Failure         *string             `json:"failure"`
	FailedOperation string              `json:"failed_operation,omitempty"`
	Controls        []*ControlResult    `json:"controls"`
	T0              float64             `json:"t0"`
	T               float64             `json:"t"`
}

// TestKeys contains the experiment results.
type TestKeys struct {
	NetworkEvents []*model.ArchivalNetworkEvent             `json:"network_events"`
	Queries       []*model.ArchivalDNSLookupResult          `json:"queries"`
	TCPConnect    []*model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Targets       []*TargetResult                           `json:"signal_targets"`

	// SignalServiceStatus is Signal's own outage beacon: uptime.signal.org resolves to
	// 127.0.0.1 when the service is up and 127.0.0.2 during an outage.
	SignalServiceStatus string `json:"signal_service_status"`

	// SignalRoots is the SHA-256 of each root used for Signal's hosts (lowercase hex of the DER):
	// the two built-in roots unless SignalCA replaced them.
	SignalRoots []string `json:"signal_roots"`

	// ControlsTruncated is true when the test's clock ran out before all controls were started.
	ControlsTruncated bool `json:"signal_controls_truncated"`

	// Circumvention holds the circumvention routes (null unless Circumvention is set).
	Circumvention []*RouteResult `json:"signal_circumvention"`
}

// ResponseSample is a bounded record of an HTTP answer (a refused upgrade, or an https_get's
// response): enough to tell Signal's own answer from the CDN's, such as a geoblock or rate-limit
// page. Header values and the body are scrubbed of IP addresses when serialized.
type ResponseSample struct {
	Status          int64                                              `json:"status"`
	Headers         map[string]model.ArchivalScrubbedMaybeBinaryString `json:"headers"`
	Body            model.ArchivalScrubbedMaybeBinaryString            `json:"body"`
	BodyIsTruncated bool                                               `json:"body_is_truncated"`
}

// newTargetResult is a target's result before it is measured.
func newTargetResult(target Target, zero time.Time) *TargetResult {
	address := ""
	return &TargetResult{Name: target.Name, Hostname: target.Hostname, Address: &address, ClientHello: target.Hello.Name,
		Method: target.Method, T0: time.Since(zero).Seconds()}
}

// markInterrupted records as interrupted every failure that ended after the test's clock ran
// out at firedAt (seconds since the start): they are the clock's, not the network's.
func (tk *TestKeys) markInterrupted(firedAt float64) {
	interrupted := netxlite.FailureInterrupted
	mark := func(failure **string, t float64) {
		if *failure != nil && t >= firedAt {
			*failure = &interrupted
		}
	}
	markTarget := func(r *TargetResult) {
		if r == nil {
			return
		}
		mark(&r.Failure, r.T)
		if r.Provisioning != nil {
			mark(&r.Provisioning.Failure, r.T)
		}
		for _, c := range r.Controls {
			mark(&c.Failure, c.T)
		}
	}
	for _, r := range tk.Targets {
		markTarget(r)
	}
	for _, route := range tk.Circumvention {
		if route == nil {
			continue
		}
		mark(&route.Failure, route.T)
		for _, r := range route.Targets {
			markTarget(r)
		}
	}
}

// mergeRecords appends src's network records to tk's.
func (tk *TestKeys) mergeRecords(src *TestKeys) {
	tk.Queries = append(tk.Queries, src.Queries...)
	tk.TCPConnect = append(tk.TCPConnect, src.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, src.TLSHandshakes...)
	tk.NetworkEvents = append(tk.NetworkEvents, src.NetworkEvents...)
}
