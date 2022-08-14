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
	// Queries contains DNS queries.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`

	// fundamentalFailure indicates that some fundamental error occurred
	// in a background task. A fundamental error is something like a programmer
	// such as a failure to parse a URL that was hardcoded in the codebase. When
	// this class of errors happens, you certainly don't want to submit the
	// resulting measurement to the OONI collector.
	fundamentalFailure error

	// mu provides mutual exclusion for accessing the test keys.
	mu *sync.Mutex
}

// AppendQueries appends to Queries.
func (tk *TestKeys) AppendQueries(v ...*model.ArchivalDNSLookupResult) {
	tk.mu.Lock()
	tk.Queries = append(tk.Queries, v...)
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
		fundamentalFailure: nil,
		mu:                 &sync.Mutex{},
	}
}

// finalize performs any delayed computation on the test keys. This function
// must be called from the measurer after all the tasks have completed.
func (tk *TestKeys) finalize() {
	// TODO: implement
}
