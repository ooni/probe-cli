package telegram

//
// TestKeys for telegram.
//
// Note: for historical reasons, we call TestKeys the JSON object
// containing the results produced by OONI experiments.
//

import (
	"errors"
	"sync"
	"syscall"

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

	// TelegramTCPBlocking indicates whether we believe DCs
	// to be blocked at the TCP/IP layer. From the spec: "If all
	// TCP connections on ports 80 and 443 to Telegram’s access
	// point IPs fail we consider Telegram to be blocked."
	TelegramTCPBlocking bool `json:"telegram_tcp_blocking"`

	// TelegramHTTPBlocking indicates whether we believe DCs
	// to be blocked at the TCP/IP layer. From the spec: "If at
	// least an HTTP request returns back a response, we
	// consider Telegram [DCs] to not be blocked."
	TelegramHTTPBlocking bool `json:"telegram_http_blocking"`

	// TelegramWebStatus is either "blocked" or "ok" and indicates
	// whether we're able to access the web.telegram.org site.
	TelegramWebStatus string `json:"telegram_web_status"`

	// TelegramWebFailure is the failure when accessing web.telegram.org
	TelegramWebFailure *string `json:"telegram_web_failure"`

	// webFailures contains the failures occurred when measuring web.telegram.org
	webFailures []error

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

// SetTelegramTCPBlocking sets the value of TelegramTCPBlocking.
func (tk *TestKeys) SetTelegramTCPBlocking(value bool) {
	tk.mu.Lock()
	tk.TelegramTCPBlocking = value
	tk.mu.Unlock()
}

// SetTelegramHTTPBlocking sets the value of TelegramHTTPBlocking.
func (tk *TestKeys) SetTelegramHTTPBlocking(value bool) {
	tk.mu.Lock()
	tk.TelegramHTTPBlocking = value
	tk.mu.Unlock()
}

// AppendWebFailure appends to the webFailures list.
func (tk *TestKeys) AppendWebFailure(err error) {
	tk.mu.Lock()
	tk.webFailures = append(tk.webFailures, err)
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
	tk := &TestKeys{
		NetworkEvents:        []*model.ArchivalNetworkEvent{},
		Queries:              []*model.ArchivalDNSLookupResult{},
		Requests:             []*model.ArchivalHTTPRequestResult{},
		TCPConnect:           []*model.ArchivalTCPConnectResult{},
		TLSHandshakes:        []*model.ArchivalTLSOrQUICHandshakeResult{},
		TelegramTCPBlocking:  false,
		TelegramHTTPBlocking: false,
		TelegramWebStatus:    "",
		TelegramWebFailure:   nil,
		webFailures:          []error{},
		fundamentalFailure:   nil,
		mu:                   &sync.Mutex{},
	}

	// "If all TCP connections on ports 80 and 443 to Telegram’s access
	// point IPs fail we consider Telegram to be blocked."
	tk.TelegramTCPBlocking = true

	// "If at least an HTTP request returns back a response, we
	// consider Telegram [DCs] to not be blocked."
	tk.TelegramHTTPBlocking = true

	// We start saying web.telegram.org is blocked and flip to okay
	// only when we notice that it's accessible.
	tk.TelegramWebStatus = "blocked"

	// We start by saying that the experiment did not actually
	// run until completion, and then flip later if needed.
	didNotRun := "telegram_did_not_run_error"
	tk.TelegramWebFailure = &didNotRun

	return tk
}

// finalize performs any delayed computation on the test keys. This function
// must be called from the measurer after all the tasks have completed.
func (tk *TestKeys) finalize() {
	var filtered []error
	for _, err := range tk.webFailures {
		if errors.Is(err, syscall.EHOSTUNREACH) || errors.Is(err, syscall.ENETUNREACH) {
			continue // skip IPv6 errors when there's no working IPv6 support
		}
		filtered = append(filtered, err)
	}
	if len(filtered) <= 0 {
		tk.TelegramWebStatus = "ok"
		tk.TelegramWebFailure = nil
		return
	}
	tk.TelegramWebStatus = "blocked"
	first := filtered[0].Error()
	tk.TelegramWebFailure = &first
}
