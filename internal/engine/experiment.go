package engine

//
// Experiment definition and implementation.
//

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const dateFormat = "2006-01-02 15:04:05"

func formatTimeNowUTC() string {
	return time.Now().UTC().Format(dateFormat)
}

// experiment implements Experiment.
type experiment struct {
	byteCounter   *bytecounter.Counter
	callbacks     model.ExperimentCallbacks
	measurer      model.ExperimentMeasurer
	report        probeservices.ReportChannel
	session       *Session
	testName      string
	testStartTime string
	testVersion   string
}

// newExperiment creates a new experiment given a measurer.
func newExperiment(sess *Session, measurer model.ExperimentMeasurer) *experiment {
	return &experiment{
		byteCounter:   bytecounter.New(),
		callbacks:     model.NewPrinterCallbacks(sess.Logger()),
		measurer:      measurer,
		session:       sess,
		testName:      measurer.ExperimentName(),
		testStartTime: formatTimeNowUTC(),
		testVersion:   measurer.ExperimentVersion(),
	}
}

// KibiBytesReceived implements Experiment.KibiBytesReceived.
func (e *experiment) KibiBytesReceived() float64 {
	return e.byteCounter.KibiBytesReceived()
}

// KibiBytesSent implements Experiment.KibiBytesSent.
func (e *experiment) KibiBytesSent() float64 {
	return e.byteCounter.KibiBytesSent()
}

// Name implements Experiment.Name.
func (e *experiment) Name() string {
	return e.testName
}

// GetSummaryKeys implements Experiment.GetSummaryKeys.
func (e *experiment) GetSummaryKeys(m *model.Measurement) (interface{}, error) {
	return e.measurer.GetSummaryKeys(m)
}

// ReportID implements Experiment.ReportID.
func (e *experiment) ReportID() string {
	if e.report == nil {
		return ""
	}
	return e.report.ReportID()
}

// experimentAsyncWrapper makes a sync experiment behave like it was async
type experimentAsyncWrapper struct {
	*experiment
}

var _ model.ExperimentMeasurerAsync = &experimentAsyncWrapper{}

// RunAsync implements ExperimentMeasurerAsync.RunAsync.
func (eaw *experimentAsyncWrapper) RunAsync(
	ctx context.Context, sess model.ExperimentSession, input string,
	callbacks model.ExperimentCallbacks) (<-chan *model.ExperimentAsyncTestKeys, error) {
	out := make(chan *model.ExperimentAsyncTestKeys)
	measurement := eaw.experiment.newMeasurement(input)
	start := time.Now()
	err := eaw.experiment.measurer.Run(ctx, eaw.session, measurement, eaw.callbacks)
	stop := time.Now()
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(out) // signal the reader we're done!
		out <- &model.ExperimentAsyncTestKeys{
			Extensions:         measurement.Extensions,
			Input:              measurement.Input,
			MeasurementRuntime: stop.Sub(start).Seconds(),
			TestKeys:           measurement.TestKeys,
			TestHelpers:        measurement.TestHelpers,
		}
	}()
	return out, nil
}

// MeasureAsync implements Experiment.MeasureAsync.
func (e *experiment) MeasureAsync(
	ctx context.Context, input string) (<-chan *model.Measurement, error) {
	err := e.session.MaybeLookupLocationContext(ctx) // this already tracks session bytes
	if err != nil {
		return nil, err
	}
	ctx = bytecounter.WithSessionByteCounter(ctx, e.session.byteCounter)
	ctx = bytecounter.WithExperimentByteCounter(ctx, e.byteCounter)
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
		defer close(out) // we need to signal the consumer we're done
		for tk := range in {
			measurement := e.newMeasurement(input)
			measurement.Extensions = tk.Extensions
			measurement.Input = tk.Input
			measurement.MeasurementRuntime = tk.MeasurementRuntime
			measurement.TestHelpers = tk.TestHelpers
			measurement.TestKeys = tk.TestKeys
			if err := model.ScrubMeasurement(measurement, e.session.ProbeIP()); err != nil {
				// If we fail to scrub the measurement then we are not going to
				// submit it. Most likely causes of error here are unlikely,
				// e.g., the TestKeys being not serializable.
				e.session.Logger().Warnf("can't scrub measurement: %s", err.Error())
				continue
			}
			out <- measurement
		}
	}()
	return out, nil
}

// MeasureWithContext implements Experiment.MeasureWithContext.
func (e *experiment) MeasureWithContext(
	ctx context.Context, input string,
) (measurement *model.Measurement, err error) {
	out, err := e.MeasureAsync(ctx, input)
	if err != nil {
		return nil, err
	}
	for m := range out {
		if measurement == nil {
			measurement = m // as documented just return the first one
		}
	}
	if measurement == nil {
		err = errors.New("experiment returned no measurements")
	}
	return
}

// SaveMeasurement implements Experiment.SaveMeasurement.
func (e *experiment) SaveMeasurement(measurement *model.Measurement, filePath string) error {
	return e.saveMeasurement(
		measurement, filePath, json.Marshal, os.OpenFile,
		func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
}

// SubmitAndUpdateMeasurementContext implements Experiment.SubmitAndUpdateMeasurementContext.
func (e *experiment) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	if e.report == nil {
		return errors.New("report is not open")
	}
	return e.report.SubmitMeasurement(ctx, measurement)
}

// newMeasurement creates a new measurement for this experiment with the given input.
func (e *experiment) newMeasurement(input string) *model.Measurement {
	utctimenow := time.Now().UTC()
	m := &model.Measurement{
		DataFormatVersion:         probeservices.DefaultDataFormatVersion,
		Input:                     model.MeasurementTarget(input),
		MeasurementStartTime:      utctimenow.Format(dateFormat),
		MeasurementStartTimeSaved: utctimenow,
		ProbeIP:                   model.DefaultProbeIP,
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
	m.AddAnnotation("architecture", runtime.GOARCH)
	return m
}

// OpenReportContext implements Experiment.OpenReportContext.
func (e *experiment) OpenReportContext(ctx context.Context) error {
	if e.report != nil {
		return nil // already open
	}
	// use custom client to have proper byte accounting
	httpClient := &http.Client{
		Transport: bytecounter.WrapHTTPTransport(
			e.session.httpDefaultTransport, // proxy is OK
			e.byteCounter,
		),
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

func (e *experiment) newReportTemplate() probeservices.ReportTemplate {
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

func (e *experiment) saveMeasurement(
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
