package dash

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/montanaflynn/stats"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestTestKeysAnalyzeWithNoData(t *testing.T) {
	tk := &TestKeys{}
	err := tk.analyze()
	if !errors.Is(err, stats.EmptyInputErr) {
		t.Fatal("expected an error here")
	}
}

func TestTestKeysAnalyzeMinPlayoutDelay(t *testing.T) {
	tk := &TestKeys{
		ReceiverData: []clientResults{
			{
				ElapsedTarget: 2,
				Elapsed:       1.4,
			},
			{
				ElapsedTarget: 2,
				Elapsed:       3.0,
			},
			{
				ElapsedTarget: 2,
				Elapsed:       1.8,
			},
		},
	}
	err := tk.analyze()
	if err != nil {
		t.Fatal(err)
	}
	if tk.Simple.MinPlayoutDelay < 0.99 || tk.Simple.MinPlayoutDelay > 1.01 {
		t.Fatal("unexpected min-playout-delay value")
	}
}

func TestTestKeysAnalyzeMedian(t *testing.T) {
	tk := &TestKeys{
		ReceiverData: []clientResults{
			{
				Rate: 1,
			},
			{
				Rate: 2,
			},
			{
				Rate: 3,
			},
		},
	}
	err := tk.analyze()
	if err != nil {
		t.Fatal(err)
	}
	if tk.Simple.MedianBitrate != 2 {
		t.Fatal("unexpected median value")
	}
}

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "dash" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.14.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasureWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	measurement := new(model.Measurement)
	m := &Measurer{}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mocks.Session{
			MockDefaultHTTPClient: func() model.HTTPClient {
				return http.DefaultClient
			},
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
			MockUserAgent: func() string {
				return "miniooni/0.1.0-dev"
			},
		},
	}
	err := m.Run(ctx, args)
	// See corresponding comment in Measurer.Run implementation to
	// understand why here it's correct to return nil.
	if !errors.Is(err, nil) {
		t.Fatal("unexpected error value")
	}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysGood(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &TestKeys{Simple: Simple{
		ConnectLatency:  1234,
		MedianBitrate:   123,
		MinPlayoutDelay: 12,
	}}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.Latency != 1234 {
		t.Fatal("invalid latency")
	}
	if sk.Bitrate != 123 {
		t.Fatal("invalid bitrate")
	}
	if sk.Delay != 12 {
		t.Fatal("invalid delay")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
