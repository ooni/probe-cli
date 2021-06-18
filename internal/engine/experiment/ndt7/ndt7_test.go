package ndt7

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "ndt" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.9.0" {
		t.Fatal("unexpected version")
	}
}

func TestDiscoverCancelledContext(t *testing.T) {
	m := new(Measurer)
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
		MockableUserAgent:  "miniooni/0.1.0-dev",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	locateResult, err := m.discover(ctx, sess)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if locateResult.Hostname != "" {
		t.Fatal("not the Hostname we expected")
	}
}

func TestDoDownloadWithCancelledContext(t *testing.T) {
	m := new(Measurer)
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
		MockableUserAgent:  "miniooni/0.1.0-dev",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	err := m.doDownload(
		ctx, sess, model.NewPrinterCallbacks(log.Log), new(TestKeys),
		"ws://host.name")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected")
	}
}

func TestDoUploadWithCancelledContext(t *testing.T) {
	m := new(Measurer)
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
		MockableUserAgent:  "miniooni/0.1.0-dev",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	err := m.doUpload(
		ctx, sess, model.NewPrinterCallbacks(log.Log), new(TestKeys),
		"ws://host.name")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected")
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	m := new(Measurer)
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
		MockableUserAgent:  "miniooni/0.1.0-dev",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel
	err := m.Run(ctx, sess, new(model.Measurement), model.NewPrinterCallbacks(log.Log))
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
}

func TestGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurement := new(model.Measurement)
	measurer := NewExperimentMeasurer(Config{})
	err := measurer.Run(
		context.Background(),
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestFailDownload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	measurer := NewExperimentMeasurer(Config{}).(*Measurer)
	measurer.preDownloadHook = func() {
		cancel()
	}
	err := measurer.Run(
		ctx,
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal(err)
	}
}

func TestFailUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	measurer := NewExperimentMeasurer(Config{noDownload: true}).(*Measurer)
	measurer.preUploadHook = func() {
		cancel()
	}
	err := measurer.Run(
		ctx,
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal(err)
	}
}

func TestDownloadJSONUnmarshalFail(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{noUpload: true}).(*Measurer)
	var seenError bool
	expected := errors.New("expected error")
	measurer.jsonUnmarshal = func(data []byte, v interface{}) error {
		seenError = true
		return expected
	}
	err := measurer.Run(
		context.Background(),
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !seenError {
		t.Fatal("did not see expected error")
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
	measurement := &model.Measurement{TestKeys: &TestKeys{Summary: Summary{
		RetransmitRate: 1,
		MSS:            2,
		MinRTT:         3,
		AvgRTT:         4,
		MaxRTT:         5,
		Ping:           6,
		Download:       7,
		Upload:         8,
	}}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.RetransmitRate != 1 {
		t.Fatal("invalid retransmitRate")
	}
	if sk.MSS != 2 {
		t.Fatal("invalid mss")
	}
	if sk.MinRTT != 3 {
		t.Fatal("invalid minRTT")
	}
	if sk.AvgRTT != 4 {
		t.Fatal("invalid minRTT")
	}
	if sk.MaxRTT != 5 {
		t.Fatal("invalid minRTT")
	}
	if sk.Ping != 6 {
		t.Fatal("invalid minRTT")
	}
	if sk.Download != 7 {
		t.Fatal("invalid minRTT")
	}
	if sk.Upload != 8 {
		t.Fatal("invalid minRTT")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
