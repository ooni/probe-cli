// Package handlers contains default modelx.Handler handlers.
package handlers

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
)

type stdoutHandler struct{}

func (stdoutHandler) OnMeasurement(m modelx.Measurement) {
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "unexpected json.Marshal failure")
	fmt.Printf("%s\n", string(data))
}

// StdoutHandler is a Handler that logs on stdout.
var StdoutHandler stdoutHandler

type noHandler struct{}

func (noHandler) OnMeasurement(m modelx.Measurement) {
}

// NoHandler is a Handler that does not print anything
var NoHandler noHandler

// SavingHandler saves the events it receives.
type SavingHandler struct {
	mu sync.Mutex
	v  []modelx.Measurement
}

// OnMeasurement implements modelx.Handler.OnMeasurement
func (sh *SavingHandler) OnMeasurement(ev modelx.Measurement) {
	sh.mu.Lock()
	sh.v = append(sh.v, ev)
	sh.mu.Unlock()
}

// Read extracts the saved events
func (sh *SavingHandler) Read() []modelx.Measurement {
	sh.mu.Lock()
	v := sh.v
	sh.v = nil
	sh.mu.Unlock()
	return v
}
