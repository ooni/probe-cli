package tlsmiddlebox

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CompleteTrace records the result of the network trace
// using a control SNI and a target SNI
type CompleteTrace struct {
	Address       string               `json:"address"`
	TCPTraceroute *IterativeTraceroute `json:"tcp_traceroute"`
	ControlTrace  *IterativeTrace      `json:"control_trace"`
	TargetTrace   *IterativeTrace      `json:"target_trace"`
}

// Trace is an iterative trace for the corresponding servername and address
type IterativeTrace struct {
	SNI        string       `json:"server_name"`
	Iterations []*Iteration `json:"iterations"`

	mu sync.Mutex
}

// Iteration is a single network iteration with variable TTL
type Iteration struct {
	TTL       int                                     `json:"ttl"`
	Handshake *model.ArchivalTLSOrQUICHandshakeResult `json:"handshake"`
}

// IterativeTraceroute is an iterative traceroute towards the address
type IterativeTraceroute struct {
	SNI        string           `json:"server_name"`
	Iterations []*ICMPIteration `json:"iterations"`

	mu sync.Mutex
}

// ICMPIteration is a single ICMP error message associated with a TTL-limited
// probe when used for traceroute
type ICMPIteration struct {
	TTL       int                             `json:"ttl"`
	ICMPError *model.ArchivalICMPErrorMessage `json:"icmp_error"`
}

// NewIterationFromHandshake returns a new iteration from a model.ArchivalTLSOrQUICHandshakeResult
func newIterationFromHandshake(ttl int, err error, soErr error, handshake *model.ArchivalTLSOrQUICHandshakeResult) *Iteration {
	if err != nil {
		return &Iteration{
			TTL: ttl,
			Handshake: &model.ArchivalTLSOrQUICHandshakeResult{
				Failure: tracex.NewFailure(err),
			},
		}
	}
	handshake.SoError = tracex.NewFailure(soErr)
	return &Iteration{
		TTL:       ttl,
		Handshake: handshake,
	}
}

// NewICMPIterationFromHandshake returns a new iteration from a model.ArchivalICMPErrorMessage
func newICMPIterationFromHandshake(ttl int, err error, soErr error, icmpError *model.ArchivalICMPErrorMessage) *ICMPIteration {
	if err != nil {
		return &ICMPIteration{
			TTL: ttl,
			ICMPError: &model.ArchivalICMPErrorMessage{
				Timeout: "yes",
			},
		}
	}

	return &ICMPIteration{
		TTL:       ttl,
		ICMPError: icmpError,
	}

}

// addIterations adds iterations to the trace
func (t *IterativeTrace) addIterations(ev ...*Iteration) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev...)
	t.mu.Unlock()
}

// addIterationsTraceroute adds iterations to the traceroute
func (t *IterativeTraceroute) addIterationsTraceroute(ev ...*ICMPIteration) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev...)
	t.mu.Unlock()
}
