package signalclient

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// newMeasurer returns a Measurer configured as Run would configure it from a target with config.
func newMeasurer(config Config) *Measurer {
	m := NewExperimentMeasurer().(*Measurer)
	m.config = config
	return m
}

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer()
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

// Run measures only with a richer-input target that carries a config.
func TestRunRequiresATarget(t *testing.T) {
	for name, c := range map[string]struct {
		target model.ExperimentTarget
		want   error
	}{
		"no target":         {nil, ErrInputRequired},
		"another type":      {&model.OOAPIURLInfo{}, ErrInvalidInputType},
		"target, no config": {&RicherInputTarget{}, ErrInvalidInputType},
	} {
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
			Target:      c.target,
		}
		if err := NewExperimentMeasurer().Run(context.Background(), args); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", name, err, c.want)
		}
	}
}
