// Package psiphon implements the psiphon network experiment. This
// implements, in particular, v0.2.0 of the spec.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-015-psiphon.md
package psiphon

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "psiphon"
	testVersion = "0.6.1"
)

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType
)

// Config contains the experiment's configuration.
type Config struct{}

// TestKeys contains the experiment's result.
type TestKeys struct {
	urlgetter.TestKeys
	MaxRuntime float64 `json:"max_runtime"`
}

// Measurer is the psiphon measurer.
type Measurer struct {
	BeforeGetHook func(g urlgetter.Getter)
}

// ExperimentName returns the experiment name
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion returns the experiment version
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) printprogress(
	ctx context.Context, wg *sync.WaitGroup,
	maxruntime int, callbacks model.ExperimentCallbacks,
) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	step := 1 / float64(maxruntime)
	var progress float64
	defer callbacks.OnProgress(1.0, "psiphon experiment complete")
	defer wg.Done()
	for {
		select {
		case <-ticker.C:
			progress += step
			callbacks.OnProgress(progress, "psiphon experiment running")
		case <-ctx.Done():
			return
		}
	}
}

// Run runs the measurement
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	const maxruntime = 300
	// obtain the richer-input target
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}

	ctx, cancel := context.WithTimeout(ctx, maxruntime*time.Second)
	var (
		wg     sync.WaitGroup
		config urlgetter.Config
	)
	wg.Add(1)
	go m.printprogress(ctx, &wg, maxruntime, callbacks)
	config.Tunnel = "psiphon" // force to use psiphon tunnel
	urlgetter.RegisterExtensions(measurement)
	targetURL := "https://www.google.com/humans.txt"
	if target.URL != "" {
		targetURL = target.URL
	}
	g := urlgetter.Getter{
		Config:  config,
		Session: sess,
		Target:  targetURL,
	}
	if m.BeforeGetHook != nil {
		m.BeforeGetHook(g)
	}
	tk, _ := g.Get(ctx) // ignore error since we have the testkeys and want to submit them
	cancel()
	wg.Wait()
	measurement.TestKeys = &TestKeys{
		TestKeys:   tk,
		MaxRuntime: maxruntime,
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}

var _ model.MeasurementSummaryKeysProvider = &TestKeys{}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	BootstrapTime float64 `json:"bootstrap_time"`
	Failure       string  `json:"failure"`
	IsAnomaly     bool    `json:"-"`
}

// MeasurementSummaryKeys implements model.MeasurementSummaryKeysProvider.
func (tk *TestKeys) MeasurementSummaryKeys() model.MeasurementSummaryKeys {
	sk := &SummaryKeys{IsAnomaly: false}
	if tk.Failure != nil {
		sk.Failure = *tk.Failure
		sk.IsAnomaly = true
	}
	sk.BootstrapTime = tk.BootstrapTime
	return sk
}

// Anomaly implements model.MeasurementSummaryKeys.
func (sk *SummaryKeys) Anomaly() bool {
	return sk.IsAnomaly
}
