package signalclient

import (
	"testing"
	"time"
)

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "signal_client" || m.ExperimentVersion() != "0.2.0" {
		t.Errorf("%s %s", m.ExperimentName(), m.ExperimentVersion())
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

func TestIdleDurationIsClamped(t *testing.T) {
	for in, want := range map[int64]time.Duration{-5: 0, 0: 0, 20: 20 * time.Second, 30: 30 * time.Second, 60: 30 * time.Second, 1 << 62: 30 * time.Second} {
		if got := idleDuration(in); got != want {
			t.Errorf("idleDuration(%d) = %v, want %v", in, got, want)
		}
	}
}
