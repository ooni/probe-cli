package engine

import (
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestNewSaverDisabled(t *testing.T) {
	saver, err := NewSaver(SaverConfig{
		Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := saver.(fakeSaver); !ok {
		t.Fatal("not the type of Saver we expected")
	}
	m := new(model.Measurement)
	if err := saver.SaveMeasurement(m); err != nil {
		t.Fatal(err)
	}
}

func TestNewSaverWithEmptyFilePath(t *testing.T) {
	saver, err := NewSaver(SaverConfig{
		Enabled:  true,
		FilePath: "",
	})
	if err == nil || err.Error() != "saver: passed an empty filepath" {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if saver != nil {
		t.Fatal("saver should be nil here")
	}
}

type FakeSaverExperiment struct {
	M        *model.Measurement
	Error    error
	FilePath string
}

func (fse *FakeSaverExperiment) SaveMeasurement(m *model.Measurement, filepath string) error {
	fse.M = m
	fse.FilePath = filepath
	return fse.Error
}

var _ SaverExperiment = &FakeSaverExperiment{}

func TestNewSaverWithFailureWhenSaving(t *testing.T) {
	expected := errors.New("mocked error")
	fse := &FakeSaverExperiment{Error: expected}
	saver, err := NewSaver(SaverConfig{
		Enabled:    true,
		FilePath:   "report.jsonl",
		Experiment: fse,
		Logger:     log.Log,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := saver.(realSaver); !ok {
		t.Fatal("not the type of saver we expected")
	}
	m := &model.Measurement{Input: "www.kernel.org"}
	if err := saver.SaveMeasurement(m); !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if diff := cmp.Diff(fse.M, m); diff != "" {
		t.Fatal(diff)
	}
	if fse.FilePath != "report.jsonl" {
		t.Fatal("passed invalid filepath")
	}
}
