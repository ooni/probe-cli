package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type FakeInputProcessorExperiment struct {
	SleepTime time.Duration
	Err       error
	M         []*model.Measurement
}

func (fipe *FakeInputProcessorExperiment) MeasureAsync(
	ctx context.Context, input string) (<-chan *model.Measurement, error) {
	if fipe.Err != nil {
		return nil, fipe.Err
	}
	if fipe.SleepTime > 0 {
		time.Sleep(fipe.SleepTime)
	}
	m := new(model.Measurement)
	// Here we add annotations to ensure that the input processor
	// is MERGING annotations as opposed to overwriting them.
	m.AddAnnotation("antani", "antani")
	m.AddAnnotation("foo", "baz") // would be bar below
	m.Input = model.MeasurementTarget(input)
	fipe.M = append(fipe.M, m)
	out := make(chan *model.Measurement)
	go func() {
		defer close(out)
		out <- m
	}()
	return out, nil
}

func TestInputProcessorMeasurementFailed(t *testing.T) {
	expected := errors.New("mocked error")
	ip := &InputProcessor{
		Callbacks: model.NewPrinterCallbacks(model.DiscardLogger),
		Experiment: NewInputProcessorExperimentWrapper(
			&FakeInputProcessorExperiment{Err: expected},
		),
		Inputs: []model.OOAPIURLInfo{{
			URL: "https://www.kernel.org/",
		}},
	}
	ctx := context.Background()
	if err := ip.Run(ctx); !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

type FakeInputProcessorSubmitter struct {
	Err error
	M   []*model.Measurement
}

func (fips *FakeInputProcessorSubmitter) Submit(
	ctx context.Context, m *model.Measurement) error {
	fips.M = append(fips.M, m)
	return fips.Err
}

func TestInputProcessorSubmissionFailed(t *testing.T) {
	fipe := &FakeInputProcessorExperiment{}
	expected := errors.New("mocked error")
	ip := &InputProcessor{
		Annotations: map[string]string{
			"foo": "bar",
		},
		Callbacks:  model.NewPrinterCallbacks(model.DiscardLogger),
		Experiment: NewInputProcessorExperimentWrapper(fipe),
		Inputs: []model.OOAPIURLInfo{{
			URL: "https://www.kernel.org/",
		}},
		Options: []string{"fake=true"},
		Submitter: NewInputProcessorSubmitterWrapper(
			&FakeInputProcessorSubmitter{Err: expected},
		),
	}
	ctx := context.Background()
	if err := ip.Run(ctx); !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(fipe.M) != 1 {
		t.Fatal("no measurements generated")
	}
	m := fipe.M[0]
	if m.Input != "https://www.kernel.org/" {
		t.Fatal("invalid input")
	}
	if len(m.Annotations) != 2 {
		t.Fatal("invalid number of annotations")
	}
	if m.Annotations["foo"] != "bar" {
		t.Fatal("invalid annotation: foo")
	}
	if m.Annotations["antani"] != "antani" {
		t.Fatal("invalid annotation: antani")
	}
	if len(m.Options) != 1 || m.Options[0] != "fake=true" {
		t.Fatal("options not set")
	}
}

type FakeInputProcessorSaver struct {
	Err error
	M   []*model.Measurement
}

func (fips *FakeInputProcessorSaver) SaveMeasurement(m *model.Measurement) error {
	fips.M = append(fips.M, m)
	return fips.Err
}

func TestInputProcessorSaveOnDiskFailed(t *testing.T) {
	expected := errors.New("mocked error")
	ip := &InputProcessor{
		Callbacks: model.NewPrinterCallbacks(model.DiscardLogger),
		Experiment: NewInputProcessorExperimentWrapper(
			&FakeInputProcessorExperiment{},
		),
		Inputs: []model.OOAPIURLInfo{{
			URL: "https://www.kernel.org/",
		}},
		Options: []string{"fake=true"},
		Saver: NewInputProcessorSaverWrapper(
			&FakeInputProcessorSaver{Err: expected},
		),
		Submitter: NewInputProcessorSubmitterWrapper(
			&FakeInputProcessorSubmitter{Err: nil},
		),
	}
	ctx := context.Background()
	if err := ip.Run(ctx); !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestInputProcessorGood(t *testing.T) {
	fipe := &FakeInputProcessorExperiment{}
	saver := &FakeInputProcessorSaver{Err: nil}
	submitter := &FakeInputProcessorSubmitter{Err: nil}
	ip := &InputProcessor{
		Callbacks:  model.NewPrinterCallbacks(model.DiscardLogger),
		Experiment: NewInputProcessorExperimentWrapper(fipe),
		Inputs: []model.OOAPIURLInfo{{
			URL: "https://www.kernel.org/",
		}, {
			URL: "https://www.slashdot.org/",
		}},
		Options:   []string{"fake=true"},
		Saver:     NewInputProcessorSaverWrapper(saver),
		Submitter: NewInputProcessorSubmitterWrapper(submitter),
	}
	ctx := context.Background()
	reason, err := ip.run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if reason != stopNormal {
		t.Fatal("terminated by max runtime!?")
	}
	if len(fipe.M) != 2 || len(saver.M) != 2 || len(submitter.M) != 2 {
		t.Fatal("not all measurements saved")
	}
	if submitter.M[0].Input != "https://www.kernel.org/" {
		t.Fatal("invalid submitter.M[0].Input")
	}
	if submitter.M[1].Input != "https://www.slashdot.org/" {
		t.Fatal("invalid submitter.M[1].Input")
	}
	if saver.M[0].Input != "https://www.kernel.org/" {
		t.Fatal("invalid saver.M[0].Input")
	}
	if saver.M[1].Input != "https://www.slashdot.org/" {
		t.Fatal("invalid saver.M[1].Input")
	}
}

func TestInputProcessorMaxRuntime(t *testing.T) {
	fipe := &FakeInputProcessorExperiment{
		SleepTime: 50 * time.Millisecond,
	}
	saver := &FakeInputProcessorSaver{Err: nil}
	submitter := &FakeInputProcessorSubmitter{Err: nil}
	ip := &InputProcessor{
		Callbacks:  model.NewPrinterCallbacks(model.DiscardLogger),
		Experiment: NewInputProcessorExperimentWrapper(fipe),
		Inputs: []model.OOAPIURLInfo{{
			URL: "https://www.kernel.org/",
		}, {
			URL: "https://www.slashdot.org/",
		}},
		MaxRuntime: 1 * time.Nanosecond,
		Options:    []string{"fake=true"},
		Saver:      NewInputProcessorSaverWrapper(saver),
		Submitter:  NewInputProcessorSubmitterWrapper(submitter),
	}
	ctx := context.Background()
	reason, err := ip.run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if reason != stopMaxRuntime {
		t.Fatal("not terminated by max runtime")
	}
}
