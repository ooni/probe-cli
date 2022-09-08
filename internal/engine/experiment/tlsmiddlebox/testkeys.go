package tlsmiddlebox

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestKeys contains the experiment results
type TestKeys struct {
	Queries        []*model.ArchivalDNSLookupResult  `json:"queries"`
	TCPConnect     []*model.ArchivalTCPConnectResult `json:"tcp_connect"`
	IterativeTrace []*CompleteTrace                  `json:"iterative_trace"`

	mu sync.Mutex
}

// NewTestKeys creates new tlsmiddlebox TestKeys
func NewTestKeys() *TestKeys {
	return &TestKeys{
		Queries:        []*model.ArchivalDNSLookupResult{},
		TCPConnect:     []*model.ArchivalTCPConnectResult{},
		IterativeTrace: []*CompleteTrace{},
	}
}

// addqueries adds []*model.ArchivalDNSLookupResut to the test keys queries
func (tk *TestKeys) addQueries(ev []*model.ArchivalDNSLookupResult) {
	tk.mu.Lock()
	tk.Queries = append(tk.Queries, ev...)
	tk.mu.Unlock()
}

// addTCPConnect adds []*model.ArchivalTCPConnectResult to the test keys TCPConnect
func (tk *TestKeys) addTCPConnect(ev []*model.ArchivalTCPConnectResult) {
	tk.mu.Lock()
	tk.TCPConnect = append(tk.TCPConnect, ev...)
	tk.mu.Unlock()
}

// addTrace adds []*CompleteTrace to the test keys Trace
func (tk *TestKeys) addTrace(ev ...*CompleteTrace) {
	tk.mu.Lock()
	tk.IterativeTrace = append(tk.IterativeTrace, ev...)
	tk.mu.Unlock()
}
