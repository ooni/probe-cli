package signalclient

import "testing"

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
