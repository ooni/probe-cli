package engine

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestCreateAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	for _, name := range AllExperiments() {
		builder, err := sess.NewExperimentBuilder(name)
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment()
		good := (exp.Name() == name)
		if !good {
			t.Fatal("unexpected experiment name")
		}
	}
}

func TestRunDASH(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("dash")
	if err != nil {
		t.Fatal(err)
	}
	if !builder.Interruptible() {
		t.Fatal("dash not marked as interruptible")
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestRunExample(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestRunNdt7(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("ndt7")
	if err != nil {
		t.Fatal(err)
	}
	if !builder.Interruptible() {
		t.Fatal("ndt7 not marked as interruptible")
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestRunPsiphon(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("psiphon")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestRunSNIBlocking(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("sni_blocking")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "kernel.org")
}

func TestRunTelegram(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("telegram")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestRunTor(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("tor")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func TestNeedsInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("web_connectivity")
	if err != nil {
		t.Fatal(err)
	}
	if builder.InputPolicy() != InputOrQueryBackend {
		t.Fatal("web_connectivity certainly needs input")
	}
}

func TestSetCallbacks(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.SetOptionAny("SleepTime", 0); err != nil {
		t.Fatal(err)
	}
	register := &registerCallbacksCalled{}
	builder.SetCallbacks(register)
	if _, err := builder.NewExperiment().MeasureWithContext(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if register.onProgressCalled == false {
		t.Fatal("OnProgress not called")
	}
}

type registerCallbacksCalled struct {
	onProgressCalled bool
}

func (c *registerCallbacksCalled) OnProgress(percentage float64, message string) {
	c.onProgressCalled = true
}

func TestCreateInvalidExperiment(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("antani")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if builder != nil {
		t.Fatal("expected a nil builder here")
	}
}

func TestMeasurementFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	if err := builder.SetOptionAny("ReturnError", true); err != nil {
		t.Fatal(err)
	}
	measurement, err := builder.NewExperiment().MeasureWithContext(context.Background(), "")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err.Error() != "mocked error" {
		t.Fatal("unexpected error type")
	}
	if measurement != nil {
		t.Fatal("expected nil measurement here")
	}
}

func TestUseOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	options, err := builder.Options()
	if err != nil {
		t.Fatal(err)
	}
	var (
		returnError bool
		message     bool
		sleepTime   bool
		other       int64
	)
	for name, option := range options {
		if name == "ReturnError" {
			returnError = true
			if option.Type != "bool" {
				t.Fatal("ReturnError is not a bool")
			}
			if option.Doc != "Toogle to return a mocked error" {
				t.Fatal("ReturnError doc is wrong")
			}
		} else if name == "Message" {
			message = true
			if option.Type != "string" {
				t.Fatal("Message is not a string")
			}
			if option.Doc != "Message to emit at test completion" {
				t.Fatal("Message doc is wrong")
			}
		} else if name == "SleepTime" {
			sleepTime = true
			if option.Type != "int64" {
				t.Fatal("SleepTime is not an int64")
			}
			if option.Doc != "Amount of time to sleep for" {
				t.Fatal("SleepTime doc is wrong")
			}
		} else {
			other++
		}
	}
	if other != 0 {
		t.Fatal("found unexpected option")
	}
	if !returnError {
		t.Fatal("did not find ReturnError option")
	}
	if !message {
		t.Fatal("did not find Message option")
	}
	if !sleepTime {
		t.Fatal("did not find SleepTime option")
	}
	if err := builder.SetOptionAny("ReturnError", true); err != nil {
		t.Fatal("cannot set ReturnError field")
	}
	if err := builder.SetOptionAny("SleepTime", 10); err != nil {
		t.Fatal("cannot set SleepTime field")
	}
	if err := builder.SetOptionAny("Message", "antani"); err != nil {
		t.Fatal("cannot set Message field")
	}
	config := builder.(*experimentBuilder).config.(*example.Config)
	if config.ReturnError != true {
		t.Fatal("config.ReturnError was not changed")
	}
	if config.SleepTime != 10 {
		t.Fatal("config.SleepTime was not changed")
	}
	if config.Message != "antani" {
		t.Fatal("config.Message was not changed")
	}
}

func TestRunHHFM(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("http_header_field_manipulation")
	if err != nil {
		t.Fatal(err)
	}
	runexperimentflow(t, builder.NewExperiment(), "")
}

func runexperimentflow(t *testing.T, experiment model.Experiment, input string) {
	ctx := context.Background()
	err := experiment.OpenReportContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if experiment.ReportID() == "" {
		t.Fatal("reportID should not be empty here")
	}
	measurement, err := experiment.MeasureWithContext(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	measurement.AddAnnotations(map[string]string{
		"probe-engine-ci": "yes",
	})
	data, err := json.Marshal(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if data == nil {
		t.Fatal("data is nil")
	}
	err = experiment.SubmitAndUpdateMeasurementContext(ctx, measurement)
	if err != nil {
		t.Fatal(err)
	}
	err = experiment.SaveMeasurement(measurement, "/tmp/experiment.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if experiment.KibiBytesSent() <= 0 {
		t.Fatal("no data sent?!")
	}
	if experiment.KibiBytesReceived() <= 0 {
		t.Fatal("no data received?!")
	}
	if _, err := experiment.GetSummaryKeys(measurement); err != nil {
		t.Fatal(err)
	}
}

func TestSaveMeasurementErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	exp := builder.NewExperiment().(*experiment)
	dirname, err := ioutil.TempDir("", "ooniprobe-engine-save-measurement")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(dirname, "report.jsonl")
	m := new(model.Measurement)
	err = exp.saveMeasurement(
		m, filename, func(v interface{}) ([]byte, error) {
			return nil, errors.New("mocked error")
		}, os.OpenFile, func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	err = exp.saveMeasurement(
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
	err = exp.saveMeasurement(
		m, filename, json.Marshal, os.OpenFile,
		func(fp *os.File, b []byte) (int, error) {
			return 0, errors.New("mocked error")
		},
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestOpenReportIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	exp := builder.NewExperiment()
	if exp.ReportID() != "" {
		t.Fatal("unexpected initial report ID")
	}
	ctx := context.Background()
	if err := exp.SubmitAndUpdateMeasurementContext(ctx, &model.Measurement{}); err == nil {
		t.Fatal("we should not be able to submit before OpenReport")
	}
	err = exp.OpenReportContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rid := exp.ReportID()
	if rid == "" {
		t.Fatal("invalid report ID")
	}
	err = exp.OpenReportContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rid != exp.ReportID() {
		t.Fatal("OpenReport is not idempotent")
	}
}

func TestOpenReportFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		},
	))
	defer server.Close()
	sess := newSessionForTestingNoBackendsLookup(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	exp := builder.NewExperiment().(*experiment)
	exp.session.selectedProbeService = &model.OOAPIService{
		Address: server.URL,
		Type:    "https",
	}
	err = exp.OpenReportContext(context.Background())
	if !strings.HasPrefix(err.Error(), "httpx: request failed") {
		t.Fatal("not the error we expected")
	}
}

func TestOpenReportNewClientFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTestingNoBackendsLookup(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	exp := builder.NewExperiment().(*experiment)
	exp.session.selectedProbeService = &model.OOAPIService{
		Address: "antani:///",
		Type:    "antani",
	}
	err = exp.OpenReportContext(context.Background())
	if err.Error() != "probe services: unsupported endpoint type" {
		t.Fatal(err)
	}
}

func TestSubmitAndUpdateMeasurementWithClosedReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTesting(t)
	defer sess.Close()
	builder, err := sess.NewExperimentBuilder("example")
	if err != nil {
		t.Fatal(err)
	}
	exp := builder.NewExperiment()
	m := new(model.Measurement)
	err = exp.SubmitAndUpdateMeasurementContext(context.Background(), m)
	if err == nil {
		t.Fatal("expected an error here")
	}
}

func TestMeasureLookupLocationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTestingNoLookups(t)
	defer sess.Close()
	exp := newExperiment(sess, new(antaniMeasurer))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // so we fail immediately
	if _, err := exp.MeasureWithContext(ctx, "xx"); err == nil {
		t.Fatal("expected an error here")
	}
}

func TestOpenReportNonHTTPS(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := newSessionForTestingNoLookups(t)
	defer sess.Close()
	sess.availableProbeServices = []model.OOAPIService{
		{
			Address: "antani",
			Type:    "mascetti",
		},
	}
	exp := newExperiment(sess, new(antaniMeasurer))
	if err := exp.OpenReportContext(context.Background()); err == nil {
		t.Fatal("expected an error here")
	}
}

type antaniMeasurer struct{}

func (am *antaniMeasurer) ExperimentName() string {
	return "antani"
}

func (am *antaniMeasurer) ExperimentVersion() string {
	return "0.1.1"
}

func (am *antaniMeasurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	return nil
}

func (am *antaniMeasurer) GetSummaryKeys(m *model.Measurement) (interface{}, error) {
	return struct {
		Failure *string `json:"failure"`
	}{}, nil
}
