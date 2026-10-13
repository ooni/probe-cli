package signalclient

import (
	"testing"
	"time"
)

// The whole measurement's budget is the sum of its phases' own limits plus a margin: 160 s with
// the default options, 360 s with the circumvention routes (ts-041 Time limits).
func TestRunBudget(t *testing.T) {
	if got := runBudget(0, false); got != 160*time.Second {
		t.Errorf("default: %v", got)
	}
	if got := runBudget(0, true); got != 360*time.Second {
		t.Errorf("with routes: %v", got)
	}
	if got := runBudget(60*time.Second, false); got != 220*time.Second {
		t.Errorf("with 60 s idle: %v", got)
	}
}

type recordedProgress struct{ values []float64 }

func (r *recordedProgress) OnProgress(p float64, _ string) { r.values = append(r.values, p) }

// Progress counts the beacon, each target, the controls and each route, and ends at 1.
func TestProgress(t *testing.T) {
	rec := &recordedProgress{}
	p := newProgress(rec, 12, 2)
	for i := 0; i < 1+12+1+2; i++ {
		p.step("")
	}
	if len(rec.values) != 16 || rec.values[15] != 1 || rec.values[0] <= 0 {
		t.Errorf("progress %v", rec.values)
	}
}
