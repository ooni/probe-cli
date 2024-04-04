package oonirun

import (
	"errors"
	"os"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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

func TestNewSaverWithFailureWhenSaving(t *testing.T) {
	filep := runtimex.Try1(os.CreateTemp("", ""))
	filename := filep.Name()
	filep.Close()
	expected := errors.New("mocked error")
	saver, err := NewSaver(SaverConfig{
		Enabled:  true,
		FilePath: filename,
		Logger:   log.Log,
	})
	if err != nil {
		t.Fatal(err)
	}
	realSaver, ok := saver.(*realSaver)
	if !ok {
		t.Fatal("not the type of saver we expected")
	}
	var (
		gotMeasurement *model.Measurement
		gotFilePath    string
	)
	realSaver.savefunc = func(measurement *model.Measurement, filePath string) error {
		gotMeasurement, gotFilePath = measurement, filePath
		return expected
	}
	m := &model.Measurement{Input: "www.kernel.org"}
	if err := saver.SaveMeasurement(m); !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if diff := cmp.Diff(m, gotMeasurement); diff != "" {
		t.Fatal(diff)
	}
	if gotFilePath != filename {
		t.Fatal("passed invalid filepath")
	}
}
