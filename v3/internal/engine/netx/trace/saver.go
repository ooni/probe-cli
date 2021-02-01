package trace

import "sync"

// The Saver saves a trace
type Saver struct {
	ops []Event
	mu  sync.Mutex
}

// Read reads and returns events inside the trace. It advances
// the read pointer so you won't see such events again.
func (s *Saver) Read() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.ops
	s.ops = nil
	return v
}

// Write adds the given event to the trace. A subsequent call
// to Read will read this event.
func (s *Saver) Write(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ops = append(s.ops, ev)
}
