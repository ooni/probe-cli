package dnsping

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestKeys contains the experiment results.
type TestKeys struct {
	Pings []*SinglePing `json:"pings"`

	// mu provides mutual exclusion
	mu sync.Mutex
}

// SinglePing contains the results of a single ping.
type SinglePing struct {
	Query            *model.ArchivalDNSLookupResult   `json:"query"`
	DelayedResponses []*model.ArchivalDNSLookupResult `json:"delayed_responses"`
}

// NewTestKeys creates new dnsping TestKeys
func NewTestKeys() *TestKeys {
	return &TestKeys{
		Pings: []*SinglePing{},
		mu:    sync.Mutex{},
	}
}

// addSinglePing adds []*SinglePing to the test keys
func (tk *TestKeys) addPings(pings []*SinglePing) {
	tk.mu.Lock()
	tk.Pings = append(tk.Pings, pings...)
	tk.mu.Unlock()
}
