package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestSaveMeasurementSuccess(t *testing.T) {
	// get temporary file where to write the measurement
	filep, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	filename := filep.Name()
	filep.Close()

	// create and fake-fill the measurement
	m := &model.Measurement{}
	ff := &testingx.FakeFiller{}
	ff.Fill(m)

	// write the measurement to disk
	if err := SaveMeasurement(m, filename); err != nil {
		t.Fatal(err)
	}

	// marshal the measurement to JSON with extra \n at the end
	expect := append(must.MarshalJSON(m), '\n')

	// read the measurement from file
	got := runtimex.Try1(os.ReadFile(filename))

	// make sure what we read matches what we expect
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestSaveMeasurementErrors(t *testing.T) {
	dirname, err := os.MkdirTemp("", "ooniprobe-engine-save-measurement")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dirname, "report.jsonl")
	m := new(model.Measurement)
	err = saveMeasurement(
		m, filename, func(v interface{}) ([]byte, error) {
			return nil, errors.New("mocked error")
		}, os.OpenFile, func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	err = saveMeasurement(
		m, filename, json.Marshal,
		func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, errors.New("mocked error")
		}, func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	err = saveMeasurement(
		m, filename, json.Marshal, os.OpenFile,
		func(fp *os.File, b []byte) (int, error) {
			return 0, errors.New("mocked error")
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}
