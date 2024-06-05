package engine

//
// Experiment definition and implementation.
//

import (
	"context"
	"errors"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/probeservices"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// experimentMutableReport is the mutable experiment.report field.
//
// We isolate this into a separate data structure to ease code management. By using this
// pattern, we don't need to be concerned with locking mutexes multiple times and it's just
// a matter of using public methods exported by this struct, which are goroutine safe.
type experimentMutableReport struct {
	mu     sync.Mutex
	report probeservices.ReportChannel
}

// Set atomically sets the report possibly overriding a previously set report.
//
// This method is goroutine safe.
func (emr *experimentMutableReport) Set(report probeservices.ReportChannel) {
	emr.mu.Lock()
	emr.report = report
	emr.mu.Unlock()
}

// Get atomically gets the report possibly returning nil.
func (emr *experimentMutableReport) Get() (report probeservices.ReportChannel) {
	emr.mu.Lock()
	report = emr.report
	emr.mu.Unlock()
	return
}

// experiment implements [model.Experiment].
type experiment struct {
	byteCounter   *bytecounter.Counter
	callbacks     model.ExperimentCallbacks
	measurer      model.ExperimentMeasurer
	mrep          *experimentMutableReport
	session       *Session
	testName      string
	testStartTime string
	testVersion   string
}

// newExperiment creates a new [*experiment] given a [model.ExperimentMeasurer].
func newExperiment(sess *Session, measurer model.ExperimentMeasurer) *experiment {
	return &experiment{
		byteCounter:   bytecounter.New(),
		callbacks:     model.NewPrinterCallbacks(sess.Logger()),
		measurer:      measurer,
		mrep:          &experimentMutableReport{},
		session:       sess,
		testName:      measurer.ExperimentName(),
		testStartTime: model.MeasurementFormatTimeNowUTC(),
		testVersion:   measurer.ExperimentVersion(),
	}
}

// KibiBytesReceived implements [model.Experiment].
func (e *experiment) KibiBytesReceived() float64 {
	return e.byteCounter.KibiBytesReceived()
}

// KibiBytesSent implements [model.Experiment].
func (e *experiment) KibiBytesSent() float64 {
	return e.byteCounter.KibiBytesSent()
}

// Name implements [model.Experiment].
func (e *experiment) Name() string {
	return e.testName
}

// ReportID implements [model.Experiment].
func (e *experiment) ReportID() string {
	report := e.mrep.Get()
	if report == nil {
		return ""
	}
	return report.ReportID()
}

// SubmitAndUpdateMeasurementContext implements [model.Experiment].
func (e *experiment) SubmitAndUpdateMeasurementContext(
	ctx context.Context, measurement *model.Measurement) error {
	report := e.mrep.Get()
	if report == nil {
		return errors.New("report is not open")
	}
	return report.SubmitMeasurement(ctx, measurement)
}

// newMeasurement creates a new measurement for this experiment with the given input.
func (e *experiment) newMeasurement(input string) *model.Measurement {
	utctimenow := time.Now().UTC()
	m := &model.Measurement{
		DataFormatVersion:         model.OOAPIReportDefaultDataFormatVersion,
		Input:                     model.MeasurementInput(input),
		MeasurementStartTime:      utctimenow.Format(model.MeasurementDateFormat),
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
	m.AddAnnotation("architecture", runtime.GOARCH)
	m.AddAnnotation("engine_name", "ooniprobe-engine")
	m.AddAnnotation("engine_version", version.Version)
	m.AddAnnotation("go_version", runtimex.BuildInfo.GoVersion)
	m.AddAnnotation("platform", e.session.Platform())
	m.AddAnnotation("vcs_modified", runtimex.BuildInfo.VcsModified)
	m.AddAnnotation("vcs_revision", runtimex.BuildInfo.VcsRevision)
	m.AddAnnotation("vcs_time", runtimex.BuildInfo.VcsTime)
	m.AddAnnotation("vcs_tool", runtimex.BuildInfo.VcsTool)
	return m
}

// OpenReportContext implements Experiment.OpenReportContext.
func (e *experiment) OpenReportContext(ctx context.Context) error {
	// handle the case where we already opened the report
	report := e.mrep.Get()
	if report != nil {
		return nil // already open
	}

	// use custom client to have proper byte accounting
	httpClient := &http.Client{
		Transport: bytecounter.WrapHTTPTransport(
			e.session.network.HTTPTransport(),
			e.byteCounter,
		),
	}
	client, err := e.session.newProbeServicesClient(ctx)
	if err != nil {
		e.session.logger.Debugf("%+v", err)
		return err
	}
	client.HTTPClient = httpClient // patch HTTP client to use

	// create the report template to open the report
	template := e.newReportTemplate()

	// attempt to open the report
	report, err = client.OpenReport(ctx, template)

	// handle the error case
	if err != nil {
		e.session.logger.Debugf("experiment: probe services error: %s", err.Error())
		return err
	}

	// on success, assign the new report
	e.mrep.Set(report)
	return nil
}

// MeasureWithContext implements [model.Experiment].
func (e *experiment) MeasureWithContext(ctx context.Context, input string) (*model.Measurement, error) {
	// Here we ensure that we have already looked up the probe location
	// information such that we correctly populate the measurement and also
	// VERY IMPORTANTLY to scrub the IP address from the measurement.
	//
	// Also, this SHOULD happen before wrapping the context for byte counting
	// since MaybeLookupLocationContext already accounts for bytes I/O.
	//
	// TODO(bassosimone,DecFox): historically we did this only for measuring
	// and not for opening a report, which probably is not correct. Because the
	// function call is idempotent, call it also when opening a report?
	if err := e.session.MaybeLookupLocationContext(ctx); err != nil {
		return nil, err
	}

	// Tweak the context such that the bytes sent and received are accounted
	// to both the session's byte counter and to the experiment's byte counter.
	ctx = bytecounter.WithSessionByteCounter(ctx, e.session.byteCounter)
	ctx = bytecounter.WithExperimentByteCounter(ctx, e.byteCounter)

	// Create a new measurement that the experiment measurer will finish filling
	// by adding the test keys etc. Please, note that, as of 2024-06-05, we're using
	// the measurement Input to provide input to an experiment. We'll probably
	// change this, when we'll have finished implementing richer input.
	measurement := e.newMeasurement(input)

	// Record when we started the experiment, to compute the runtime.
	start := time.Now()

	// Prepare the arguments for the experiment measurer
	args := &model.ExperimentArgs{
		Callbacks:   e.callbacks,
		Measurement: measurement,
		Session:     e.session,
	}

	// Invoke the measurer. Conventionally, an error being returned here
	// indicates that something went wrong during the measurement. For example,
	// it could be that the user provided us with a malformed input. In case
	// there's censorship, by all means the experiment should return a nil error
	// and fill the measurement accordingly.
	err := e.measurer.Run(ctx, args)

	// Record when the experiment finished running.
	stop := time.Now()

	// Handle the case where there was a fundamental error.
	if err != nil {
		return nil, err
	}

	// Make sure we record the measurement runtime.
	measurement.MeasurementRuntime = stop.Sub(start).Seconds()

	// Scub the measurement removing the probe IP addr from it. We are 100% sure we know
	// our own IP addr, since we called MaybeLookupLocation above. Obviously, we aren't
	// going to submit the measurement in case we can't scrub it, so we just return an error
	// if this specific corner case happens.
	//
	// TODO(bassosimone,DecFox): a dual stack client MAY be such that we discover its IPv4
	// address but the IPv6 address is present inside the measurement. We should ensure that
	// we improve our discovering capabilities to also cover this specific case.
	if err := model.ScrubMeasurement(measurement, e.session.ProbeIP()); err != nil {
		e.session.Logger().Warnf("can't scrub measurement: %s", err.Error())
		return nil, err
	}

	// We're all good! Let us return the measurement to the caller, which will
	// addtionally take care that we're submitting it, if needed.
	return measurement, nil
}

func (e *experiment) newReportTemplate() model.OOAPIReportTemplate {
	return model.OOAPIReportTemplate{
		DataFormatVersion: model.OOAPIReportDefaultDataFormatVersion,
		Format:            model.OOAPIReportDefaultFormat,
		ProbeASN:          e.session.ProbeASNString(),
		ProbeCC:           e.session.ProbeCC(),
		SoftwareName:      e.session.SoftwareName(),
		SoftwareVersion:   e.session.SoftwareVersion(),
		TestName:          e.testName,
		TestStartTime:     e.testStartTime,
		TestVersion:       e.testVersion,
	}
}

// ExperimentMeasurementSummaryKeysNotImplemented is the [model.MeasurementSummary] we use when
// the experiment TestKeys do not provide an implementation of [model.MeasurementSummary].
type ExperimentMeasurementSummaryKeysNotImplemented struct{}

var _ model.MeasurementSummaryKeys = &ExperimentMeasurementSummaryKeysNotImplemented{}

// IsAnomaly implements MeasurementSummary.
func (*ExperimentMeasurementSummaryKeysNotImplemented) Anomaly() bool {
	return false
}

// MeasurementSummaryKeys returns the [model.MeasurementSummaryKeys] associated with a given measurement.
func MeasurementSummaryKeys(m *model.Measurement) model.MeasurementSummaryKeys {
	if tk, ok := m.TestKeys.(model.MeasurementSummaryKeysProvider); ok {
		return tk.MeasurementSummaryKeys()
	}
	return &ExperimentMeasurementSummaryKeysNotImplemented{}
}
