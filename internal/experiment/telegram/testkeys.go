package telegram

//
// TestKeys for telegram.
//
// Note: for historical reasons, we call TestKeys the JSON object
// containing the results produced by OONI experiments.
//

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestKeys contains the results produced by telegram.
type TestKeys struct {
	// NetworkEvents contains network events.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// Queries contains DNS lookup results.
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
	tk.Requests = append(tk.Requests, v...)
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

// SetFundamentalFailure implements TestKeys.
func (tk *TestKeys) SetFundamentalFailure(err error) {
	tk.mu.Lock()
	tk.fundamentalFailure = err
	tk.mu.Unlock()
}

// FundamentalFailure implements TestKeys.
func (tk *TestKeys) FundamentalFailure() error {
	tk.mu.Lock()
	err := tk.fundamentalFailure
	tk.mu.Unlock()
	return err
}

// NewTestKeys creates a new instance of TestKeys.
func NewTestKeys() *TestKeys {
	// TODO: here you should initialize all the fields
	return &TestKeys{
		fundamentalFailure: nil,
		mu:                 &sync.Mutex{},
	}
}
