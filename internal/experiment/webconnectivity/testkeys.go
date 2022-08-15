package webconnectivity

//
// TestKeys for web_connectivity.
//
// Note: for historical reasons, we call TestKeys the JSON object
// containing the results produced by OONI experiments.
//

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestKeys contains the results produced by web_connectivity.
type TestKeys struct {
	// NetworkEvents contains network events.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// Queries contains DNS queries.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`

	// Requests contains HTTP results.
	Requests []*model.ArchivalHTTPRequestResult `json:"requests"`

	// TCPConnect contains TCP connect results.
	TCPConnect []*model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// TLSHandshakes contains TLS handshakes results.
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`

	// fundamentalFailure indicates that some fundamental error occurred
	// in a background task. A fundamental error is something like a programmer
	// such as a failure to parse a URL that was hardcoded in the codebase. When
	// this class of errors happens, you certainly don't want to submit the
	// resulting measurement to the OONI collector.
	fundamentalFailure error

	// mu provides mutual exclusion for accessing the test keys.
	mu *sync.Mutex
}

// AppendNetworkEvents appends to NetworkEvents.
func (tk *TestKeys) AppendNetworkEvents(v ...*model.ArchivalNetworkEvent) {
	tk.mu.Lock()
	tk.NetworkEvents = append(tk.NetworkEvents, v...)
	tk.mu.Unlock()
}

// AppendQueries appends to Queries.
func (tk *TestKeys) AppendQueries(v ...*model.ArchivalDNSLookupResult) {
	tk.mu.Lock()
	tk.Queries = append(tk.Queries, v...)
	tk.mu.Unlock()
}

// AppendRequests appends to Requests.
func (tk *TestKeys) AppendRequests(v ...*model.ArchivalHTTPRequestResult) {
	tk.mu.Lock()
	// Implementation note: append at the front since the most recent
	// request must be at the beginning of the list.
	tk.Requests = append(v, tk.Requests...)
	tk.mu.Unlock()
}

// AppendTCPConnectResults appends to TCPConnect.
func (tk *TestKeys) AppendTCPConnectResults(v ...*model.ArchivalTCPConnectResult) {
	tk.mu.Lock()
	tk.TCPConnect = append(tk.TCPConnect, v...)
	tk.mu.Unlock()
}

// AppendTLSHandshakes appends to TLSHandshakes.
func (tk *TestKeys) AppendTLSHandshakes(v ...*model.ArchivalTLSOrQUICHandshakeResult) {
	tk.mu.Lock()
	tk.TLSHandshakes = append(tk.TLSHandshakes, v...)
	tk.mu.Unlock()
}

// SetFundamentalFailure sets the value of fundamentalFailure.
func (tk *TestKeys) SetFundamentalFailure(err error) {
	tk.mu.Lock()
	tk.fundamentalFailure = err
	tk.mu.Unlock()
}

// NewTestKeys creates a new instance of TestKeys.
func NewTestKeys() *TestKeys {
	// TODO: here you should initialize all the fields
	return &TestKeys{
		NetworkEvents:      []*model.ArchivalNetworkEvent{},
		Queries:            []*model.ArchivalDNSLookupResult{},
		Requests:           []*model.ArchivalHTTPRequestResult{},
		TCPConnect:         []*model.ArchivalTCPConnectResult{},
		TLSHandshakes:      []*model.ArchivalTLSOrQUICHandshakeResult{},
		fundamentalFailure: nil,
		mu:                 &sync.Mutex{},
	}
}

// finalize performs any delayed computation on the test keys. This function
// must be called from the measurer after all the tasks have completed.
func (tk *TestKeys) finalize() {
	// TODO(bassosimone): set final webconnectivity flags
	// TODO(bassosimone): sort requests correctly
}
