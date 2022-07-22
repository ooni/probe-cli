package tlsmiddlebox

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// TODO(DecFox): Synchronise types with the proposed experiment spec

// CompleteTrace records the result of the network trace
// using a control SNI and a target SNI
type CompleteTrace struct {
	Address      string `json:"address"`
	ControlTrace *Trace `json:"control_trace"`
	TargetTrace  *Trace `json:"target_trace"`
}

// Trace is an iterative trace for the corresponding servername and address
type Trace struct {
	SNI        string       `json:"server_name"`
	Iterations []*Iteration `json:"iterations"`

	mu sync.Mutex
}

// Iteration is a single network iteration with variable TTL
type Iteration struct {
	TTL       int                                     `json:"ttl"`
	Handshake *model.ArchivalTLSOrQUICHandshakeResult `json:"handshake"`
}

// NewIterationFromHandshake returns a new iteration from a model.ArchivalTLSOrQUICHandshakeResult
func newIterationFromHandshake(ttl int, err error, handshake *model.ArchivalTLSOrQUICHandshakeResult) *Iteration {
	if err != nil {
		return &Iteration{
			TTL: ttl,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: tracex.NewFailure(err),
			},
		}
	}
	return &Iteration{
		TTL:       ttl,
		Handshake: handshake,
	}
}

// addIterations adds iterations to the trace
func (t *Trace) addIterations(ev *Iteration) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev)
	t.mu.Unlock()
}
