package probeservices

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func makeMeasurement(rt model.OOAPIReportTemplate, ID string) model.Measurement {
	return model.Measurement{
		DataFormatVersion:    model.OOAPIReportDefaultDataFormatVersion,
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
		TestKeys:             map[string]any{"failure": nil},
		TestName:             rt.TestName,
		TestStartTime:        rt.TestStartTime,
		TestVersion:          rt.TestVersion,
	}
}

func newReportTemplateForTesting() model.OOAPIReportTemplate {
	return model.OOAPIReportTemplate{
		DataFormatVersion: model.OOAPIReportDefaultDataFormatVersion,
		Format:            model.OOAPIReportDefaultFormat,
		ProbeASN:          "AS117",
		ProbeCC:           "IT",
		SoftwareName:      "ooniprobe-engine",
		SoftwareVersion:   "0.1.0",
		TestName:          "dummy",
		TestStartTime:     "2019-10-28 12:51:06",
		TestVersion:       "0.1.0",
	}
}

func TestNewReportTemplate(t *testing.T) {
	// create a measurement with minimal fields
	m := &model.Measurement{
		ProbeASN:        "AS117",
		ProbeCC:         "IT",
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.1.0",
		TestName:        "dummy",
		TestStartTime:   "2019-10-28 12:51:06",
		TestVersion:     "0.1.0",
	}

	// convert the measurement to a report template
	rt := NewReportTemplate(m)

	// define expectations
	expect := newReportTemplateForTesting()

	// make sure they are equal
	if diff := cmp.Diff(expect, rt); diff != "" {
		t.Fatal(diff)
	}
}

func TestReportLifecycle(t *testing.T) {
	// First, let's check whether we can get a response from the real OONI backend.
	t.Run("is working as intended with the real backend", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}

		// create the client and report template for testing
		client := newclient()
		template := newReportTemplateForTesting()

		// open the report
		report, err := client.OpenReport(context.Background(), template)

		// we expect to be able to open the report
		if err != nil {
			t.Fatal(err)
		}

		// make a measurement out of the report template
		measurement := makeMeasurement(template, report.ReportID())

		// make sure we can submit this measurement within the report, which we really
		// expect to succeed since we created the measurement from the template
		if report.CanSubmit(&measurement) != true {
			t.Fatal("report should be able to submit this measurement")
		}

		// attempt to submit the measurement to the backend, which should succeed
		// since we've just opened a report for it
		if err = report.SubmitMeasurement(context.Background(), &measurement); err != nil {
			t.Fatal(err)
		}

		// additionally make sure we edited the measurement report ID to
		// contain the correct report ID used to submit
		if measurement.ReportID != report.ReportID() {
			t.Fatal("report ID mismatch")
		}
	})

	// Now let's construct a test server that returns a valid response and try
	// to communicate with such a test server successfully and with errors

	t.Run("is working as intended with a local test server", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONICollector{}

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state)
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client so we speak with our local server rather than the true backend
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// open the report
		report, err := client.OpenReport(context.Background(), template)

		// we expect to be able to open the report
		if err != nil {
			t.Fatal(err)
		}

		// make a measurement out of the report template
		measurement := makeMeasurement(template, report.ReportID())

		// make sure we can submit this measurement within the report, which we really
		// expect to succeed since we created the measurement from the template
		if report.CanSubmit(&measurement) != true {
			t.Fatal("report should be able to submit this measurement")
		}

		// attempt to submit the measurement to the backend, which should succeed
		// since we've just opened a report for it
		if err = report.SubmitMeasurement(context.Background(), &measurement); err != nil {
			t.Fatal(err)
		}

		// additionally make sure we edited the measurement report ID to
		// contain the correct report ID used to submit
		if measurement.ReportID != report.ReportID() {
			t.Fatal("report ID mismatch")
		}
	})

	t.Run("opening a report fails with an error when the connection is reset", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// open the report
		report, err := client.OpenReport(context.Background(), template)

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil report here
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("opening a report fails with an error when the response is not JSON parsable", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{`))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// open the report
		report, err := client.OpenReport(context.Background(), template)

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil report here
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("opening a report correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// open the report
		report, err := client.OpenReport(context.Background(), template)

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}

		// we expect a nil report here
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("updating a report fails with an error when the connection is reset", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// create the reportChan
		rc := reportChan{
			ID:     "xxx-xxx-xxx-xxx",
			client: *client,
			tmpl:   template,
		}

		// update the report
		err := rc.SubmitMeasurement(context.Background(), &model.Measurement{})

		// we do expect an error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("updating a report fails with an error when the response is not JSON parsable", func(t *testing.T) {
		// create quick and dirty server to serve the response
		srv := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{`))
		}))
		defer srv.Close()

		// create a probeservices client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// create the reportChan
		rc := reportChan{
			ID:     "xxx-xxx-xxx-xxx",
			client: *client,
			tmpl:   template,
		}

		// update the report
		err := rc.SubmitMeasurement(context.Background(), &model.Measurement{})

		// we do expect an error
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("updating a report correctly handles the case where the URL is unparseable", func(t *testing.T) {
		// create a probeservices client
		client := newclient()

		// override the URL to be unparseable
		client.BaseURL = "\t\t\t"

		// create the report template used for testing
		template := newReportTemplateForTesting()

		// create the reportChan
		rc := reportChan{
			ID:     "xxx-xxx-xxx-xxx",
			client: *client,
			tmpl:   template,
		}

		// update the report
		err := rc.SubmitMeasurement(context.Background(), &model.Measurement{})

		// we do expect an error
		if err == nil || err.Error() != `parse "\t\t\t": net/url: invalid control character in URL` {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("we cannot open a report with invalid data format version", func(t *testing.T) {
		// create client and default template
		client := newclient()
		template := newReportTemplateForTesting()

		// set a wrong data format version to test whether OpenReport would fail.
		template.DataFormatVersion = "0.1.0"

		// attempt to open the report
		report, err := client.OpenReport(context.Background(), template)

		// we expect the error to indicate the data format version is wrong
		if !errors.Is(err, ErrUnsupportedDataFormatVersion) {
			t.Fatal("not the error we expected", err)
		}

		// ancillary check: make sure report is nil
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("we cannot open a report with invalid data serialization format", func(t *testing.T) {
		// create client and default template
		client := newclient()
		template := newReportTemplateForTesting()

		// set a wrong data serialization format to test whether OpenReport would fail.
		template.Format = "yaml"

		// attempt to open the report
		report, err := client.OpenReport(context.Background(), template)

		// we expect the error to indicate the data format version is wrong
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Fatal("not the error we expected", err)
		}

		// ancillary check: make sure report is nil
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("we cannot open a report if the server doesn't support JSON", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONICollector{}

		// override the open report response to claim we only support YAML
		state.EditOpenReportResponse = func(resp *model.OOAPICollectorOpenResponse) {
			resp.SupportedFormats = []string{"yaml"}
		}

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state)
		defer srv.Close()

		// create template and client
		template := newReportTemplateForTesting()
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// attempt to open a report
		report, err := client.OpenReport(context.Background(), template)

		if !errors.Is(err, ErrJSONFormatNotSupported) {
			t.Fatal("expected an error here")
		}
		if report != nil {
			t.Fatal("expected a nil report here")
		}
	})

	t.Run("we cannot submit using the wrong experiment name", func(t *testing.T) {
		// create state for emulating the OONI backend
		state := &testingx.OONICollector{}

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state)
		defer srv.Close()

		// create template and client
		template := newReportTemplateForTesting()
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// attempt to open a report
		report, err := client.OpenReport(context.Background(), template)

		// we expect to see a success here
		if err != nil {
			t.Fatal(err)
		}

		// create a measurement to submit
		measurement := makeMeasurement(template, report.ReportID())

		// set the wrong test name to see if we can actually submit it
		measurement.TestName = "antani"

		// we expect to not be able to submit the measurement
		if report.CanSubmit(&measurement) != false {
			t.Fatal("report should not be able to submit this measurement")
		}
	})

	t.Run("end-to-end test where we verify requests and responses", func(t *testing.T) {
		// create the template
		template := newReportTemplateForTesting()

		// define the reportID we'll force
		reportID := "xxx-xx-xx-xxx"

		// create the measurement
		measurement := makeMeasurement(template, reportID)

		// create state for emulating the OONI backend
		state := &testingx.OONICollector{}

		// make sure we receive the exact report template we're sending
		state.ValidateReportTemplate = func(rt *model.OOAPIReportTemplate) error {
			if diff := cmp.Diff(&template, rt); diff != "" {
				return errors.New(diff)
			}
			return nil
		}

		// make sure we override the report ID
		state.EditOpenReportResponse = func(resp *model.OOAPICollectorOpenResponse) {
			resp.ReportID = reportID
		}

		// make sure we receive the exact measurement we're sending
		state.ValidateMeasurement = func(meas *model.Measurement) error {
			if diff := cmp.Diff(&measurement, meas); diff != "" {
				return errors.New(diff)
			}
			return nil
		}

		// define the measurement UID to expect
		measurementUID := "x-y-z-a-b-c"

		// make sure we override the measurement UID
		state.EditUpdateResponse = func(resp *model.OOAPICollectorUpdateResponse) {
			resp.MeasurementUID = measurementUID
		}

		// expose the state via HTTP
		srv := testingx.MustNewHTTPServer(state)
		defer srv.Close()

		// create client
		client := newclient()

		// override the HTTP client
		client.HTTPClient = &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				URL := runtimex.Try1(url.Parse(srv.URL))
				req.URL.Scheme = URL.Scheme
				req.URL.Host = URL.Host
				return http.DefaultClient.Do(req)
			},
			MockCloseIdleConnections: func() {
				http.DefaultClient.CloseIdleConnections()
			},
		}

		// attempt to open a report
		report, err := client.OpenReport(context.Background(), template)

		// we expect to be successful here
		if err != nil {
			t.Fatal(err)
		}

		// make sure the report ID is correct
		if report.ReportID() != reportID {
			t.Fatal("got unexpected reportID value", report.ReportID())
		}

		// make sure we can submit this measurement within the report, which we really
		// expect to succeed since we created the measurement from the template
		if report.CanSubmit(&measurement) != true {
			t.Fatal("report should be able to submit this measurement")
		}

		// attempt to submit the measurement to the backend, which should succeed
		// since we've just opened a report for it
		if err = report.SubmitMeasurement(context.Background(), &measurement); err != nil {
			t.Fatal(err)
		}

		// additionally make sure we edited the measurement report ID to
		// contain the correct report ID used to submit
		if measurement.ReportID != report.ReportID() {
			t.Fatal("report ID mismatch")
		}
	})
}

type RecordingReportChannel struct {
	tmpl model.OOAPIReportTemplate
	m    []*model.Measurement
	mu   sync.Mutex
}

func (rrc *RecordingReportChannel) CanSubmit(m *model.Measurement) bool {
	return reflect.DeepEqual(NewReportTemplate(m), rrc.tmpl)
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
	ctx context.Context, rt model.OOAPIReportTemplate,
) (ReportChannel, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	rrc := &RecordingReportChannel{tmpl: rt}
	rro.mu.Lock()
	defer rro.mu.Unlock()
	rro.channels = append(rro.channels, rrc)
	return rrc, nil
}

func makeMeasurementWithoutTemplate(testName string) *model.Measurement {
	return &model.Measurement{
		DataFormatVersion:    model.OOAPIReportDefaultDataFormatVersion,
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
		TestKeys:             map[string]any{"failure": nil},
		TestName:             testName,
		TestStartTime:        "2018-11-01 15:33:17",
		TestVersion:          "0.1.0",
	}
}

func TestSubmitterLifecyle(t *testing.T) {
	rro := &RecordingReportOpener{}
	submitter := NewSubmitter(rro, log.Log)
	ctx := context.Background()
	m1 := makeMeasurementWithoutTemplate("example")
	if err := submitter.Submit(ctx, m1); err != nil {
		t.Fatal(err)
	}
	m2 := makeMeasurementWithoutTemplate("example")
	if err := submitter.Submit(ctx, m2); err != nil {
		t.Fatal(err)
	}
	m3 := makeMeasurementWithoutTemplate("example_extended")
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
	submitter := NewSubmitter(rro, log.Log)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	m1 := makeMeasurementWithoutTemplate("example")
	if err := submitter.Submit(ctx, m1); !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	m2 := makeMeasurementWithoutTemplate("example")
	if err := submitter.Submit(ctx, m2); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	m3 := makeMeasurementWithoutTemplate("example_extended")
	if err := submitter.Submit(ctx, m3); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(rro.channels) != 0 {
		t.Fatal("unexpected number of channels")
	}
}
