package probeservices_test

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
)

type fakeTestKeys struct {
	Failure *string `json:"failure"`
}

func makeMeasurement(rt probeservices.ReportTemplate, ID string) model.Measurement {
	return model.Measurement{
		DataFormatVersion:    probeservices.DefaultDataFormatVersion,
		ID:                   "bdd20d7a-bba5-40dd-a111-9863d7908572",
		MeasurementRuntime:   5.0565230846405,
		MeasurementStartTime: "2018-11-01 15:33:20",
		ProbeIP:              "1.2.3.4",
		ProbeASN:             rt.ProbeASN,
		ProbeCC:              rt.ProbeCC,
		ReportID:             ID,
		ResolverASN:          "AS15169",
		ResolverIP:           "8.8.8.8",
		ResolverNetworkName:  "Google LLC",
		SoftwareName:         rt.SoftwareName,
		SoftwareVersion:      rt.SoftwareVersion,
		TestKeys:             fakeTestKeys{Failure: nil},
		TestName:             rt.TestName,
		TestStartTime:        rt.TestStartTime,
		TestVersion:          rt.TestVersion,
	}
}

func TestNewReportTemplate(t *testing.T) {
	m := &model.Measurement{
		ProbeASN:        "AS117",
		ProbeCC:         "IT",
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.1.0",
		TestName:        "dummy",
		TestStartTime:   "2019-10-28 12:51:06",
		TestVersion:     "0.1.0",
	}
	rt := probeservices.NewReportTemplate(m)
	expect := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS117",
		ProbeCC:           "IT",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	if diff := cmp.Diff(expect, rt); diff != "" {
		t.Fatal(diff)
	}
}

func TestReportLifecycle(t *testing.T) {
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if err != nil {
		t.Fatal(err)
	}
	measurement := makeMeasurement(template, report.ReportID())
	if report.CanSubmit(&measurement) != true {
		t.Fatal("report should be able to submit this measurement")
	}
	if err = report.SubmitMeasurement(ctx, &measurement); err != nil {
		t.Fatal(err)
	}
	if measurement.ReportID != report.ReportID() {
		t.Fatal("report ID mismatch")
	}
}

func TestReportLifecycleWrongExperiment(t *testing.T) {
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if err != nil {
		t.Fatal(err)
	}
	measurement := makeMeasurement(template, report.ReportID())
	measurement.TestName = "antani"
	if report.CanSubmit(&measurement) != false {
		t.Fatal("report should not be able to submit this measurement")
	}
}

func TestOpenReportInvalidDataFormatVersion(t *testing.T) {
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: "0.1.0",
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if !errors.Is(err, probeservices.ErrUnsupportedDataFormatVersion) {
		t.Fatal("not the error we expected")
	}
	if report != nil {
		t.Fatal("expected a nil report here")
	}
}

func TestOpenReportInvalidFormat(t *testing.T) {
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            "yaml",
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if !errors.Is(err, probeservices.ErrUnsupportedFormat) {
		t.Fatal("not the error we expected")
	}
	if report != nil {
		t.Fatal("expected a nil report here")
	}
}

func TestJSONAPIClientCreateFailure(t *testing.T) {
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	client.BaseURL = "\t" // breaks the URL parser
	report, err := client.OpenReport(ctx, template)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if report != nil {
		t.Fatal("expected a nil report here")
	}
}

func TestOpenResponseNoJSONSupport(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.Write([]byte(`{"ID":"abc","supported_formats":["yaml"]}`))
		}),
	)
	defer server.Close()
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	client.BaseURL = server.URL
	report, err := client.OpenReport(ctx, template)
	if !errors.Is(err, probeservices.ErrJSONFormatNotSupported) {
		t.Fatal("expected an error here")
	}
	if report != nil {
		t.Fatal("expected a nil report here")
	}
}

func TestEndToEnd(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RequestURI == "/report" {
				w.Write([]byte(`{"report_id":"_id","supported_formats":["json"]}`))
				return
			}
			if r.RequestURI == "/report/_id" {
				data, err := ioutil.ReadAll(r.Body)
				if err != nil {
					panic(err)
				}
				sdata, err := ioutil.ReadFile("../testdata/collector-expected.jsonl")
				if err != nil {
					panic(err)
				}
				if diff := cmp.Diff(data, sdata); diff != "" {
					panic(diff)
				}
				w.Write([]byte(`{"measurement_id":"e00c584e6e9e5326"}`))
				return
			}
			if r.RequestURI == "/report/_id/close" {
				w.Write([]byte(`{}`))
				return
			}
			panic(r.RequestURI)
		}),
	)
	defer server.Close()
	ctx := context.Background()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2018-11-01 15:33:17",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	client.BaseURL = server.URL
	report, err := client.OpenReport(ctx, template)
	if err != nil {
		t.Fatal(err)
	}
	measurement := makeMeasurement(template, report.ReportID())
	if err = report.SubmitMeasurement(ctx, &measurement); err != nil {
		t.Fatal(err)
	}
}

type RecordingReportChannel struct {
	tmpl probeservices.ReportTemplate
	m    []*model.Measurement
	mu   sync.Mutex
}

func (rrc *RecordingReportChannel) CanSubmit(m *model.Measurement) bool {
	return reflect.DeepEqual(probeservices.NewReportTemplate(m), rrc.tmpl)
}

func (rrc *RecordingReportChannel) SubmitMeasurement(ctx context.Context, m *model.Measurement) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	rrc.mu.Lock()
	defer rrc.mu.Unlock()
	rrc.m = append(rrc.m, m)
	return nil
}

func (rrc *RecordingReportChannel) Close(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	rrc.mu.Lock()
	defer rrc.mu.Unlock()
	return nil
}

func (rrc *RecordingReportChannel) ReportID() string {
	return ""
}

type RecordingReportOpener struct {
	channels []*RecordingReportChannel
	mu       sync.Mutex
}

func (rro *RecordingReportOpener) OpenReport(
	ctx context.Context, rt probeservices.ReportTemplate,
) (probeservices.ReportChannel, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	rrc := &RecordingReportChannel{tmpl: rt}
	rro.mu.Lock()
	defer rro.mu.Unlock()
	rro.channels = append(rro.channels, rrc)
	return rrc, nil
}

func TestOpenReportCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately abort
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if report != nil {
		t.Fatal("expected nil report here")
	}
}

func TestSubmitMeasurementCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	template := probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          "AS0",
		ProbeCC:           "ZZ",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
	client := newclient()
	report, err := client.OpenReport(ctx, template)
	if err != nil {
		t.Fatal(err)
	}
	measurement := makeMeasurement(template, report.ReportID())
	if report.CanSubmit(&measurement) != true {
		t.Fatal("report should be able to submit this measurement")
	}
	cancel() // cause submission to fail
	err = report.SubmitMeasurement(ctx, &measurement)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if measurement.ReportID != "" {
		t.Fatal("report ID should be empty here")
	}
}

func makeMeasurementWithoutTemplate(failure, testName string) *model.Measurement {
	return &model.Measurement{
		DataFormatVersion:    probeservices.DefaultDataFormatVersion,
		ID:                   "bdd20d7a-bba5-40dd-a111-9863d7908572",
		MeasurementRuntime:   5.0565230846405,
		MeasurementStartTime: "2018-11-01 15:33:20",
		ProbeIP:              "1.2.3.4",
		ProbeASN:             "AS123",
		ProbeCC:              "IT",
		ReportID:             "",
		ResolverASN:          "AS15169",
		ResolverIP:           "8.8.8.8",
		ResolverNetworkName:  "Google LLC",
		SoftwareName:         "miniooni",
		SoftwareVersion:      "0.1.0-dev",
		TestKeys:             fakeTestKeys{Failure: &failure},
		TestName:             testName,
		TestStartTime:        "2018-11-01 15:33:17",
		TestVersion:          "0.1.0",
	}
}

func TestSubmitterLifecyle(t *testing.T) {
	rro := &RecordingReportOpener{}
	submitter := probeservices.NewSubmitter(rro, log.Log)
	ctx := context.Background()
	m1 := makeMeasurementWithoutTemplate("antani", "example")
	if err := submitter.Submit(ctx, m1); err != nil {
		t.Fatal(err)
	}
	m2 := makeMeasurementWithoutTemplate("mascetti", "example")
	if err := submitter.Submit(ctx, m2); err != nil {
		t.Fatal(err)
	}
	m3 := makeMeasurementWithoutTemplate("antani", "example_extended")
	if err := submitter.Submit(ctx, m3); err != nil {
		t.Fatal(err)
	}
	if len(rro.channels) != 2 {
		t.Fatal("unexpected number of channels")
	}
	if len(rro.channels[0].m) != 2 {
		t.Fatal("unexpected number of measurements in first channel")
	}
	if len(rro.channels[1].m) != 1 {
		t.Fatal("unexpected number of measurements in second channel")
	}
}

func TestSubmitterCannotOpenNewChannel(t *testing.T) {
	rro := &RecordingReportOpener{}
	submitter := probeservices.NewSubmitter(rro, log.Log)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	m1 := makeMeasurementWithoutTemplate("antani", "example")
	if err := submitter.Submit(ctx, m1); !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	m2 := makeMeasurementWithoutTemplate("mascetti", "example")
	if err := submitter.Submit(ctx, m2); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	m3 := makeMeasurementWithoutTemplate("antani", "example_extended")
	if err := submitter.Submit(ctx, m3); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(rro.channels) != 0 {
		t.Fatal("unexpected number of channels")
	}
}
