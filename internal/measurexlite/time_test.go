package measurexlite

import (
	"sync"
	"testing"
	"time"
)

// timeDeterministic implements time.Now in a deterministic fashion
// such that every time.Time call returns a moment in time that occurs
// one second after the configured zeroTime.
type timeDeterministic struct {
	counter  time.Duration
	mu       sync.Mutex
	zeroTime time.Time
}

// Now is like time.Now but more deterministic.
func (td *timeDeterministic) Now() time.Time {
	td.mu.Lock()
	offset := td.counter
	td.counter += time.Second
	td.mu.Unlock()
	return td.zeroTime.Add(offset)
}

func TestTimeDeterministic(t *testing.T) {
	td := &timeDeterministic{
		counter:  0,
		mu:       sync.Mutex{},
		zeroTime: time.Now(),
	}
	t0 := td.Now()
	if !t0.Equal(td.zeroTime) {
		t.Fatal("invalid t0 value")
	}
	t1 := td.Now()
	if t1.Sub(t0) != time.Second {
		t.Fatal("invalid t1 value")
	}
	t2 := td.Now()
	if t2.Sub(t1) != time.Second {
		t.Fatal("invalid t2 value")
	}
}
