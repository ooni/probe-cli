package tlsmiddlebox

import "sync"

// CompleteTrace records the result of the network trace
// using a passed SNI and a target SNI
type CompleteTrace struct {
	PassTrace   *TraceEvent `json:"pass_trace"`
	TargetTrace *TraceEvent `json:"target_trace"`
}

// IterEvent records the result of a handshake in a single iteration
type IterEvent struct {
	Failure  *string `json:"failure"`
	TTL      int     `json:"ttl"`
	Duration int64   `json:"duration"`
}

// TraceEvent records all the iterations along with other
// information for a single SNI
type TraceEvent struct {
	Address    string       `json:"address"`
	SNI        string       `json:"servername"`
	TLSVersion string       `json:"tls_version"`
	Iterations []*IterEvent `json:"iterations"`

	mu sync.Mutex
}

func (t *TraceEvent) AddIterations(ev []*IterEvent) {
	t.mu.Lock()
	t.Iterations = append(t.Iterations, ev...)
	t.mu.Unlock()
}
