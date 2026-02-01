package urlgetter

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// TestKeys contains the experiment test keys.
type TestKeys struct {
	// The following fields are part of the typical JSON emitted by OONI.
	Agent           string                                    `json:"agent"`
	BootstrapTime   float64                                   `json:"bootstrap_time,omitempty"`
	DNSCache        []string                                  `json:"dns_cache,omitempty"`
	FailedOperation optional.Value[string]                    `json:"failed_operation"`
	Failure         optional.Value[string]                    `json:"failure"`
	NetworkEvents   []*model.ArchivalNetworkEvent             `json:"network_events"`
	Queries         []*model.ArchivalDNSLookupResult          `json:"queries"`
	Requests        []*model.ArchivalHTTPRequestResult        `json:"requests"`
	SOCKSProxy      string                                    `json:"socksproxy,omitempty"`
	TCPConnect      []*model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	TLSHandshakes   []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Tunnel          string                                    `json:"tunnel,omitempty"`

	// mu provides mutual exclusion.
	mu sync.Mutex
}

var _ RunnerTestKeys = &TestKeys{}

// AppendNetworkEvents implements RunnerTestKeys.
func (tk *TestKeys) AppendNetworkEvents(values ...*model.ArchivalNetworkEvent) {
	tk.mu.Lock()
	tk.NetworkEvents = append(tk.NetworkEvents, values...)
	tk.mu.Unlock()
}

// AppendQueries implements RunnerTestKeys.
func (tk *TestKeys) AppendQueries(values ...*model.ArchivalDNSLookupResult) {
	tk.mu.Lock()
	tk.Queries = append(tk.Queries, values...)
	tk.mu.Unlock()
}

// PrependRequests implements RunnerTestKeys.
func (tk *TestKeys) PrependRequests(values ...*model.ArchivalHTTPRequestResult) {
	tk.mu.Lock()
	// Implementation note: append at the front since the most recent
	// request must be at the beginning of the list.
	tk.Requests = append(values, tk.Requests...)
	tk.mu.Unlock()
}

// AppendTCPConnect implements RunnerTestKeys.
func (tk *TestKeys) AppendTCPConnect(values ...*model.ArchivalTCPConnectResult) {
	tk.mu.Lock()
	tk.TCPConnect = append(tk.TCPConnect, values...)
	tk.mu.Unlock()
}

// AppendTLSHandshakes implements RunnerTestKeys.
func (tk *TestKeys) AppendTLSHandshakes(values ...*model.ArchivalTLSOrQUICHandshakeResult) {
	tk.mu.Lock()
	tk.TLSHandshakes = append(tk.TLSHandshakes, values...)
	tk.mu.Unlock()
}

// MaybeSetFailedOperation implements RunnerTestKeys.
func (tk *TestKeys) MaybeSetFailedOperation(operation string) {
	tk.mu.Lock()
	if tk.FailedOperation.IsNone() {
		tk.FailedOperation = optional.Some(operation)
	}
	tk.mu.Unlock()
}

// MaybeSetFailure implements RunnerTestKeys.
func (tk *TestKeys) MaybeSetFailure(failure string) {
	tk.mu.Lock()
	if tk.Failure.IsNone() {
		tk.Failure = optional.Some(failure)
	}
	tk.mu.Unlock()
}
