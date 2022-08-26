package webconnectivity

//
// TestKeys for web_connectivity.
//
// Note: for historical reasons, we call TestKeys the JSON object
// containing the results produced by OONI experiments.
//

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// TestKeys contains the results produced by web_connectivity.
type TestKeys struct {
	// NetworkEvents contains network events.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// DNSWhoami contains results of using the DNS whoami functionality for the
	// possibly cleartext resolvers that we're using.
	DNSWoami *DNSWhoamiInfo `json:"x_dns_whoami"`

	// DoH contains ancillary observations collected by DoH resolvers.
	DoH *TestKeysDoH `json:"x_doh"`

	// Do53 contains ancillary observations collected by Do53 resolvers.
	Do53 *TestKeysDo53 `json:"x_do53"`

	// Queries contains DNS queries.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`

	// Requests contains HTTP results.
	Requests []*model.ArchivalHTTPRequestResult `json:"requests"`

	// TCPConnect contains TCP connect results.
	TCPConnect []*model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// TLSHandshakes contains TLS handshakes results.
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`

	// ControlRequest is the control request we sent.
	ControlRequest *webconnectivity.ControlRequest `json:"x_control_request"`

	// Control contains the TH's response.
	Control *webconnectivity.ControlResponse `json:"control"`

	// ControlFailure contains the failure of the control experiment.
	ControlFailure *string `json:"control_failure"`

	// DNSFlags contains DNS analysis flags.
	DNSFlags int64 `json:"x_dns_flags"`

	// DNSExperimentFailure indicates whether there was a failure in any
	// of the DNS experiments we performed.
	DNSExperimentFailure *string `json:"dns_experiment_failure"`

	// DNSConsistency indicates whether there is consistency between
	// the TH's DNS results and the probe's DNS results.
	DNSConsistency string `json:"dns_consistency"`

	// BlockingFlags contains blocking flags.
	BlockingFlags int64 `json:"x_blocking_flags"`

	// BodyLength match tells us whether the body length matches.
	BodyLengthMatch *bool `json:"body_length_match"`

	// HeadersMatch tells us whether the headers match.
	HeadersMatch *bool `json:"headers_match"`

	// StatusCodeMatch tells us whether the status code matches.
	StatusCodeMatch *bool `json:"status_code_match"`

	// TitleMatch tells us whether the title matches.
	TitleMatch *bool `json:"title_match"`

	// Blocking indicates the reason for blocking. This is notoriously a bad
	// type because it can be one of the following values:
	//
	// - "tcp_ip"
	// - "dns"
	// - "http-diff"
	// - "http-failure"
	// - false
	// - null
	//
	// In addition to having a ~bad type, this field has the issue that it
	// reduces the reason for blocking to an enum, whereas it's a set of flags,
	// hence we introduced the x_blocking_flags field.
	Blocking any `json:"blocking"`

	// Accessible indicates whether the resource is accessible. Possible
	// values for this field are: nil, true, and false.
	Accessible any `json:"accessible"`

	// fundamentalFailure indicates that some fundamental error occurred
	// in a background task. A fundamental error is something like a programmer
	// such as a failure to parse a URL that was hardcoded in the codebase. When
	// this class of errors happens, you certainly don't want to submit the
	// resulting measurement to the OONI collector.
	fundamentalFailure error

	// mu provides mutual exclusion for accessing the test keys.
	mu *sync.Mutex
}

// DNSWhoamiInfoEntry contains an entry for DNSWhoamiInfo.
type DNSWhoamiInfoEntry struct {
	// Address is the IP address
	Address string `json:"address"`
}

// DNSWhoamiInfo contains info about DNS whoami.
type DNSWhoamiInfo struct {
	// SystemV4 contains results related to the system resolver using IPv4.
	SystemV4 []DNSWhoamiInfoEntry `json:"system_v4"`

	// UDPv4 contains results related to an UDP resolver using IPv4.
	UDPv4 map[string][]DNSWhoamiInfoEntry `json:"udp_v4"`
}

// TestKeysDoH contains ancillary observations collected using DoH (e.g., the
// DNS lookups, TCP connects, TLS handshakes caused by given DoH lookups).
//
// They are on a separate hierarchy to simplify processing.
type TestKeysDoH struct {
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
}

// TestKeysDo53 contains ancillary observations collected using Do53.
//
// They are on a separate hierarchy to simplify processing.
type TestKeysDo53 struct {
	// NetworkEvents contains network events.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// Queries contains DNS queries.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`
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

// SetControlRequest sets the value of controlRequest.
func (tk *TestKeys) SetControlRequest(v *webconnectivity.ControlRequest) {
	tk.mu.Lock()
	tk.ControlRequest = v
	tk.mu.Unlock()
}

// SetControl sets the value of Control.
func (tk *TestKeys) SetControl(v *webconnectivity.ControlResponse) {
	tk.mu.Lock()
	tk.Control = v
	tk.mu.Unlock()
}

// SetControlFailure sets the value of controlFailure.
func (tk *TestKeys) SetControlFailure(err error) {
	tk.mu.Lock()
	tk.ControlFailure = tracex.NewFailure(err)
	tk.mu.Unlock()
}

// SetFundamentalFailure sets the value of fundamentalFailure.
func (tk *TestKeys) SetFundamentalFailure(err error) {
	tk.mu.Lock()
	tk.fundamentalFailure = err
	tk.mu.Unlock()
}

// WithTestKeysDoH calls the given function with the mutex locked passing to
// it as argument the pointer to the DoH field.
func (tk *TestKeys) WithTestKeysDoH(f func(*TestKeysDoH)) {
	tk.mu.Lock()
	f(tk.DoH)
	tk.mu.Unlock()
}

// WithTestKeysDo53 calls the given function with the mutex locked passing to
// it as argument the pointer to the Do53 field.
func (tk *TestKeys) WithTestKeysDo53(f func(*TestKeysDo53)) {
	tk.mu.Lock()
	f(tk.Do53)
	tk.mu.Unlock()
}

// WithDNSWhoami calls the given function with the mutex locked passing to
// it as argument the pointer to the DNSWhoami field.
func (tk *TestKeys) WithDNSWhoami(fun func(*DNSWhoamiInfo)) {
	tk.mu.Lock()
	fun(tk.DNSWoami)
	tk.mu.Unlock()
}

// NewTestKeys creates a new instance of TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		NetworkEvents: []*model.ArchivalNetworkEvent{},
		DNSWoami: &DNSWhoamiInfo{
			SystemV4: []DNSWhoamiInfoEntry{},
			UDPv4:    map[string][]DNSWhoamiInfoEntry{},
		},
		DoH: &TestKeysDoH{
			NetworkEvents: []*model.ArchivalNetworkEvent{},
			Queries:       []*model.ArchivalDNSLookupResult{},
			Requests:      []*model.ArchivalHTTPRequestResult{},
			TCPConnect:    []*model.ArchivalTCPConnectResult{},
			TLSHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{},
		},
		Do53: &TestKeysDo53{
			NetworkEvents: []*model.ArchivalNetworkEvent{},
			Queries:       []*model.ArchivalDNSLookupResult{},
		},
		Queries:              []*model.ArchivalDNSLookupResult{},
		Requests:             []*model.ArchivalHTTPRequestResult{},
		TCPConnect:           []*model.ArchivalTCPConnectResult{},
		TLSHandshakes:        []*model.ArchivalTLSOrQUICHandshakeResult{},
		Control:              nil,
		ControlFailure:       nil,
		DNSFlags:             0,
		DNSExperimentFailure: nil,
		DNSConsistency:       "",
		BlockingFlags:        0,
		BodyLengthMatch:      nil,
		HeadersMatch:         nil,
		StatusCodeMatch:      nil,
		TitleMatch:           nil,
		Blocking:             nil,
		Accessible:           nil,
		ControlRequest:       nil,
		fundamentalFailure:   nil,
		mu:                   &sync.Mutex{},
	}
}

// Finalize performs any delayed computation on the test keys. This function
// must be called from the measurer after all the tasks have completed.
func (tk *TestKeys) Finalize(logger model.Logger) {
	tk.analysisToplevel(logger)
}
