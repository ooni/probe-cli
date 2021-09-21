// Package engine contains the engine API.
package engine

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const dateFormat = "2006-01-02 15:04:05"

func formatTimeNowUTC() string {
	return time.Now().UTC().Format(dateFormat)
}

// Experiment is an experiment instance.
type Experiment struct {
	byteCounter   *bytecounter.Counter
	callbacks     model.ExperimentCallbacks
	measurer      model.ExperimentMeasurer
	report        probeservices.ReportChannel
	session       *Session
	testName      string
	testStartTime string
	testVersion   string
}

// NewExperiment creates a new experiment given a measurer. The preferred
// way to create an experiment is the ExperimentBuilder. Though this function
// allows the programmer to create a custom, external experiment.
func NewExperiment(sess *Session, measurer model.ExperimentMeasurer) *Experiment {
	return &Experiment{
		byteCounter:   bytecounter.New(),
		callbacks:     model.NewPrinterCallbacks(sess.Logger()),
		measurer:      measurer,
		session:       sess,
		testName:      measurer.ExperimentName(),
		testStartTime: formatTimeNowUTC(),
		testVersion:   measurer.ExperimentVersion(),
	}
}

// KibiBytesReceived accounts for the KibiBytes received by the HTTP clients
// managed by this session so far, including experiments.
func (e *Experiment) KibiBytesReceived() float64 {
	return e.byteCounter.KibiBytesReceived()
}

// KibiBytesSent is like KibiBytesReceived but for the bytes sent.
func (e *Experiment) KibiBytesSent() float64 {
	return e.byteCounter.KibiBytesSent()
}

// Name returns the experiment name.
func (e *Experiment) Name() string {
	return e.testName
}

// GetSummaryKeys returns a data structure containing a
// summary of the test keys for probe-cli.
func (e *Experiment) GetSummaryKeys(m *model.Measurement) (interface{}, error) {
	return e.measurer.GetSummaryKeys(m)
}

// OpenReport is an idempotent method to open a report. We assume that
// you have configured the available probe services, either manually or
// through using the session's MaybeLookupBackends method.
func (e *Experiment) OpenReport() (err error) {
	return e.OpenReportContext(context.Background())
}

// ReportID returns the open reportID, if we have opened a report
// successfully before, or an empty string, otherwise.
func (e *Experiment) ReportID() string {
	if e.report == nil {
		return ""
	}
	return e.report.ReportID()
}

// Measure performs a measurement with input. We assume that you have
// configured the available test helpers, either manually or by calling
// the session's MaybeLookupBackends() method.
//
// Deprecated: This function will return just the first measurement
// returned by the experiments that implement the model.ExperimentRunnerAsync
// interface. All the other measurements will be lost. To get all the
// measurements returned by such experiments, use MeasureAsync.
func (e *Experiment) Measure(input string) (*model.Measurement, error) {
	return e.MeasureWithContext(context.Background(), input)
}

// experimentAsyncWrapper makes a sync experiment behave like it was async
type experimentAsyncWrapper struct {
	*Experiment
}

var _ model.ExperimentMeasurerAsync = &experimentAsyncWrapper{}

// RunAsync implements ExperimentMeasurerAsync.RunAsync.
func (eaw *experimentAsyncWrapper) RunAsync(
	ctx context.Context, sess model.ExperimentSession, input string,
	callbacks model.ExperimentCallbacks) (<-chan *model.ExperimentAsyncTestKeys, error) {
	out := make(chan *model.ExperimentAsyncTestKeys)
	measurement := eaw.newMeasurement(input)
	start := time.Now()
	err := eaw.measurer.Run(ctx, eaw.session, measurement, eaw.callbacks)
	stop := time.Now()
	if err != nil {
		return nil, err
	}
	go func() {
		out <- &model.ExperimentAsyncTestKeys{
			Extensions:         measurement.Extensions,
			MeasurementRuntime: stop.Sub(start).Seconds(),
			TestKeys:           measurement.TestKeys,
		}
		close(out)
	}()
	return out, nil
}

func (e *Experiment) MeasureAsync(
	ctx context.Context, input string) (<-chan *model.Measurement, error) {
	err := e.session.MaybeLookupLocationContext(ctx) // this already tracks session bytes
	if err != nil {
		return nil, err
	}
	ctx = dialer.WithSessionByteCounter(ctx, e.session.byteCounter)
	ctx = dialer.WithExperimentByteCounter(ctx, e.byteCounter)
	var async model.ExperimentMeasurerAsync
	if v, okay := e.measurer.(model.ExperimentMeasurerAsync); okay {
		async = v
	} else {
		async = &experimentAsyncWrapper{e}
	}
	in, err := async.RunAsync(ctx, e.session, input, e.callbacks)
	if err != nil {
		return nil, err
	}
	out := make(chan *model.Measurement)
	go func() {
		defer close(out)
		for tk := range in {
			measurement := e.newMeasurement(input)
			measurement.Extensions = tk.Extensions
			measurement.MeasurementRuntime = tk.MeasurementRuntime
			measurement.TestKeys = tk.TestKeys
			if err := measurement.Scrub(e.session.ProbeIP()); err != nil {
				continue
			}
			out <- measurement
		}
	}()
	return out, nil
}

// MeasureWithContext is like Measure but with context.
//
// Deprecated: This function will return just the first measurement
// returned by the experiments that implement the model.ExperimentRunnerAsync
// interface. All the other measurements will be lost. To get all the
// measurements returned by such experiments, use MeasureAsync.
func (e *Experiment) MeasureWithContext(
	ctx context.Context, input string,
) (measurement *model.Measurement, err error) {
	out, err := e.MeasureAsync(ctx, input)
	if err != nil {
		return nil, err
	}
	for m := range out {
		if measurement == nil {
			measurement = m // as documented
		}
	}
	if measurement == nil {
		err = errors.New("experiment returned no measurements")
	}
	return
}

// SaveMeasurement saves a measurement on the specified file path.
func (e *Experiment) SaveMeasurement(measurement *model.Measurement, filePath string) error {
	return e.saveMeasurement(
		measurement, filePath, json.Marshal, os.OpenFile,
		func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
}

// SubmitAndUpdateMeasurement submits a measurement and updates the
// fields whose value has changed as part of the submission.
func (e *Experiment) SubmitAndUpdateMeasurement(measurement *model.Measurement) error {
	return e.SubmitAndUpdateMeasurementContext(context.Background(), measurement)
}

// SubmitAndUpdateMeasurementContext submits a measurement and updates the
// fields whose value has changed as part of the submission.
func (e *Experiment) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	if e.report == nil {
		return errors.New("Report is not open")
	}
	return e.report.SubmitMeasurement(ctx, measurement)
}

func (e *Experiment) newMeasurement(input string) *model.Measurement {
	utctimenow := time.Now().UTC()
	m := &model.Measurement{
		DataFormatVersion:         probeservices.DefaultDataFormatVersion,
		Input:                     model.MeasurementTarget(input),
		MeasurementStartTime:      utctimenow.Format(dateFormat),
		MeasurementStartTimeSaved: utctimenow,
		ProbeIP:                   geolocate.DefaultProbeIP,
		ProbeASN:                  e.session.ProbeASNString(),
		ProbeCC:                   e.session.ProbeCC(),
		ProbeNetworkName:          e.session.ProbeNetworkName(),
		ReportID:                  e.ReportID(),
		ResolverASN:               e.session.ResolverASNString(),
		ResolverIP:                e.session.ResolverIP(),
		ResolverNetworkName:       e.session.ResolverNetworkName(),
		SoftwareName:              e.session.SoftwareName(),
		SoftwareVersion:           e.session.SoftwareVersion(),
		TestName:                  e.testName,
		TestStartTime:             e.testStartTime,
		TestVersion:               e.testVersion,
	}
	m.AddAnnotation("engine_name", "ooniprobe-engine")
	m.AddAnnotation("engine_version", version.Version)
	m.AddAnnotation("platform", e.session.Platform())
	return m
}

// OpenReportContext will open a report using the given context
// to possibly limit the lifetime of this operation.
func (e *Experiment) OpenReportContext(ctx context.Context) error {
	if e.report != nil {
		return nil // already open
	}
	// use custom client to have proper byte accounting
	httpClient := &http.Client{
		Transport: &httptransport.ByteCountingTransport{
			RoundTripper: e.session.httpDefaultTransport, // proxy is OK
			Counter:      e.byteCounter,
		},
	}
	client, err := e.session.NewProbeServicesClient(ctx)
	if err != nil {
		e.session.logger.Debugf("%+v", err)
		return err
	}
	client.HTTPClient = httpClient // patch HTTP client to use
	template := e.newReportTemplate()
	e.report, err = client.OpenReport(ctx, template)
	if err != nil {
		e.session.logger.Debugf("experiment: probe services error: %s", err.Error())
		return err
	}
	return nil
}

func (e *Experiment) newReportTemplate() probeservices.ReportTemplate {
	return probeservices.ReportTemplate{
		DataFormatVersion: probeservices.DefaultDataFormatVersion,
		Format:            probeservices.DefaultFormat,
		ProbeASN:          e.session.ProbeASNString(),
		ProbeCC:           e.session.ProbeCC(),
		SoftwareName:      e.session.SoftwareName(),
		SoftwareVersion:   e.session.SoftwareVersion(),
		TestName:          e.testName,
		TestStartTime:     e.testStartTime,
		TestVersion:       e.testVersion,
	}
}

func (e *Experiment) saveMeasurement(
	measurement *model.Measurement, filePath string,
	marshal func(v interface{}) ([]byte, error),
	openFile func(name string, flag int, perm os.FileMode) (*os.File, error),
	write func(fp *os.File, b []byte) (n int, err error),
) error {
	data, err := marshal(measurement)
	if err != nil {
		return err
	}
	data = append(data, byte('\n'))
	filep, err := openFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := write(filep, data); err != nil {
		return err
	}
	return filep.Close()
}
