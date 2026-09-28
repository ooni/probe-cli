package tlsmiddlebox

import (
	"errors"
	"fmt"
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

func errorChain(err error) []model.FailureChainData {
	if err == nil {
		return nil
	}

	var chain []model.FailureChainData

	for err != nil {
		chain = append(chain, model.FailureChainData{
			Type:  fmt.Sprintf("%T", err),
			Error: err.Error(),
		})

		err = errors.Unwrap(err)
	}

	return chain
}

// NewIterationFromHandshake returns a new iteration from a model.ArchivalTLSOrQUICHandshakeResult
func newIterationFromHandshake(ttl int, err error, soErr error, handshake *model.ArchivalTLSOrQUICHandshakeResult) *Iteration {
	if err != nil {
		if handshake != nil {
			handshake = &model.ArchivalTLSOrQUICHandshakeResult{}
		}

		handshake.Failure = tracex.NewFailure(err)
		handshake.FailureChain = errorChain(err)

		return &Iteration{
			TTL:       ttl,
			Handshake: handshake,
		}
	}
	handshake.SoError = tracex.NewFailure(soErr)
	return &Iteration{
		TTL:       ttl,
		Handshake: handshake,
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
