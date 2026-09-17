package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperiment(t *testing.T) {
	t.Run("KibiBytesReceived", func(t *testing.T) {
		expected := 1.0
		e := &Experiment{
			MockKibiBytesReceived: func() float64 {
				return expected
			},
		}
		if e.KibiBytesReceived() != expected {
			t.Fatal("unexpected result")
		}
	})

	t.Run("KibiBytesSent", func(t *testing.T) {
		expected := 1.0
		e := &Experiment{
			MockKibiBytesSent: func() float64 {
				return expected
			},
		}
		if e.KibiBytesSent() != expected {
			t.Fatal("unexpected result")
		}
	})

	t.Run("Name", func(t *testing.T) {
		expected := "antani"
		e := &Experiment{
			MockName: func() string {
				return expected
			},
		}
		if e.Name() != expected {
			t.Fatal("unexpected result")
		}
	})

	t.Run("ReportID", func(t *testing.T) {
		expect := "xyz"
		e := &Experiment{
			MockReportID: func() string {
				return expect
			},
		}
		if e.ReportID() != expect {
			t.Fatal("invalid value")
		}
	})

	t.Run("MeasureWithContext", func(t *testing.T) {
		expected := errors.New("mocked err")
		e := &Experiment{
			MockMeasureWithContext: func(
				ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
				return nil, expected
			},
		}
		out, err := e.MeasureWithContext(
			context.Background(),
			model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("https://www.example.com/"),
		)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if out != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("SaveMeasurement", func(t *testing.T) {
		expected := errors.New("mocked err")
		e := &Experiment{
			MockSaveMeasurement: func(measurement *model.Measurement, filePath string) error {
				return expected
			},
		}
		err := e.SaveMeasurement(&model.Measurement{}, "x")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("SubmitAndUpdateMeasurementContext", func(t *testing.T) {
		expected := errors.New("mocked err")
		e := &Experiment{
			MockSubmitAndUpdateMeasurementContext: func(ctx context.Context, measurement *model.Measurement) (string, error) {
				return "", expected
			},
		}
		_, err := e.SubmitAndUpdateMeasurementContext(context.Background(), &model.Measurement{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("OpenReportContext", func(t *testing.T) {
		expected := errors.New("mocked err")
		e := &Experiment{
			MockOpenReportContext: func(ctx context.Context) error {
				return expected
			},
		}
		err := e.OpenReportContext(context.Background())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})
}
