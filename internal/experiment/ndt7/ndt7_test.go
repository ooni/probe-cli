package ndt7

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "ndt" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.10.0" {
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
	if locateResult != nil {
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
	if err == nil || err.Error() != netxlite.FailureInterrupted {
		t.Fatal("not the error we expected", err)
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
	if err == nil || err.Error() != netxlite.FailureInterrupted {
		t.Fatal("not the error we expected", err)
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
	meas := &model.Measurement{}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: meas,
		Session:     sess,
	}
	err := m.Run(ctx, args)
	// Here we get nil because we still want to submit this measurement
	if !errors.Is(err, nil) {
		t.Fatal("not the error we expected")
	}
	if meas.TestKeys == nil {
		t.Fatal("nil test keys")
	}
	tk := meas.TestKeys.(*TestKeys)
	if tk.Failure == nil || *tk.Failure != netxlite.FailureInterrupted {
		t.Fatal("unexpected tk.Failure")
	}
}

func TestGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurement := new(model.Measurement)
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
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
	meas := &model.Measurement{}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: meas,
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
	}
	err := measurer.Run(ctx, args)
	// We expect a nil failure here because we want to submit anyway
	// a measurement that failed to connect to m-lab.
	if err != nil {
		t.Fatal(err)
	}
	if meas.TestKeys == nil {
		t.Fatal("expected non-nil TestKeys here")
	}
	tk := meas.TestKeys.(*TestKeys)
	if tk.Failure == nil || *tk.Failure != netxlite.FailureInterrupted {
		t.Fatal("unexpected tk.Failure")
	}
}

func TestFailUpload(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	measurer := NewExperimentMeasurer(Config{noDownload: true}).(*Measurer)
	measurer.preUploadHook = func() {
		cancel()
	}
	meas := &model.Measurement{}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: meas,
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
	}
	err := measurer.Run(ctx, args)
	// Here we expect a nil error because we want to submit this measurement
	if err != nil {
		t.Fatal(err)
	}
	if meas.TestKeys == nil {
		t.Fatal("expected non-nil tk.TestKeys here")
	}
	tk := meas.TestKeys.(*TestKeys)
	if tk.Failure == nil || *tk.Failure != netxlite.FailureInterrupted {
		t.Fatal("unexpected tk.Failure value")
	}
}

func TestDownloadJSONUnmarshalFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewExperimentMeasurer(Config{noUpload: true}).(*Measurer)
	var seenError bool
	expected := errors.New("expected error")
	measurer.jsonUnmarshal = func(data []byte, v interface{}) error {
		seenError = true
		return expected
	}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{},
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
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
