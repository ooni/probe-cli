// Package tasks implements tasks run using the oonimkall API.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	failureIPLookup              = "failure.ip_lookup"
	failureASNLookup             = "failure.asn_lookup"
	failureCCLookup              = "failure.cc_lookup"
	failureMeasurement           = "failure.measurement"
	failureMeasurementSubmission = "failure.measurement_submission"
	failureReportCreate          = "failure.report_create"
	failureResolverLookup        = "failure.resolver_lookup"
	failureStartup               = "failure.startup"
	measurement                  = "measurement"
	statusEnd                    = "status.end"
	statusGeoIPLookup            = "status.geoip_lookup"
	statusMeasurementDone        = "status.measurement_done"
	statusMeasurementStart       = "status.measurement_start"
	statusMeasurementSubmission  = "status.measurement_submission"
	statusProgress               = "status.progress"
	statusQueued                 = "status.queued"
	statusReportCreate           = "status.report_create"
	statusResolverLookup         = "status.resolver_lookup"
	statusStarted                = "status.started"
)

// Run runs the task specified by settings.Name until completion. This is the
// top-level API that should be called by oonimkall.
func Run(ctx context.Context, settings *Settings, out chan<- *Event) {
	r := NewRunner(settings, out)
	r.Run(ctx)
}

// Runner runs a specific task
type Runner struct {
	emitter             *EventEmitter
	maybeLookupLocation func(*engine.Session) error
	out                 chan<- *Event
	settings            *Settings
}

// NewRunner creates a new task runner
func NewRunner(settings *Settings, out chan<- *Event) *Runner {
	return &Runner{
		emitter:  NewEventEmitter(settings.DisabledEvents, out),
		out:      out,
		settings: settings,
	}
}

// FailureInvalidVersion is the failure returned when Version is invalid
const FailureInvalidVersion = "invalid Settings.Version number"

func (r *Runner) hasUnsupportedSettings(logger *ChanLogger) bool {
	if r.settings.Version < 1 {
		r.emitter.EmitFailureStartup(FailureInvalidVersion)
		return true
	}
	return false
}

func (r *Runner) newsession(ctx context.Context, logger *ChanLogger) (*engine.Session, error) {
	kvstore, err := kvstore.NewFS(r.settings.StateDir)
	if err != nil {
		return nil, err
	}

	// TODO(bassosimone): write tests for this functionality
	// See https://github.com/ooni/probe/issues/1465.
	var proxyURL *url.URL
	if r.settings.Proxy != "" {
		var err error
		proxyURL, err = url.Parse(r.settings.Proxy)
		if err != nil {
			return nil, err
		}
	}

	config := engine.SessionConfig{
		KVStore:         kvstore,
		Logger:          logger,
		ProxyURL:        proxyURL,
		SoftwareName:    r.settings.Options.SoftwareName,
		SoftwareVersion: r.settings.Options.SoftwareVersion,
		TempDir:         r.settings.TempDir,
		TunnelDir:       r.settings.TunnelDir,
	}
	if r.settings.Options.ProbeServicesBaseURL != "" {
		config.AvailableProbeServices = []model.Service{{
			Type:    "https",
			Address: r.settings.Options.ProbeServicesBaseURL,
		}}
	}
	return engine.NewSession(ctx, config)
}

func (r *Runner) contextForExperiment(
	ctx context.Context, builder *engine.ExperimentBuilder,
) context.Context {
	if builder.Interruptible() {
		return ctx
	}
	return context.Background()
}

type runnerCallbacks struct {
	emitter *EventEmitter
}

func (cb *runnerCallbacks) OnProgress(percentage float64, message string) {
	cb.emitter.Emit(statusProgress, EventStatusProgress{
		Percentage: 0.4 + (percentage * 0.6), // open report is 40%
		Message:    message,
	})
}

// Run runs the runner until completion. The context argument controls
// when to stop when processing multiple inputs, as well as when to stop
// experiments explicitly marked as interruptible.
func (r *Runner) Run(ctx context.Context) {
	logger := NewChanLogger(r.emitter, r.settings.LogLevel, r.out)
	r.emitter.Emit(statusQueued, eventEmpty{})
	if r.hasUnsupportedSettings(logger) {
		return
	}
	r.emitter.Emit(statusStarted, eventEmpty{})
	sess, err := r.newsession(ctx, logger)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	endEvent := new(eventStatusEnd)
	defer func() {
		sess.Close()
		r.emitter.Emit(statusEnd, endEvent)
	}()

	builder, err := sess.NewExperimentBuilder(r.settings.Name)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	logger.Info("Looking up OONI backends... please, be patient")
	if err := sess.MaybeLookupBackends(); err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.1, "contacted bouncer")

	logger.Info("Looking up your location... please, be patient")
	maybeLookupLocation := r.maybeLookupLocation
	if maybeLookupLocation == nil {
		maybeLookupLocation = func(sess *engine.Session) error {
			return sess.MaybeLookupLocation()
		}
	}
	if err := maybeLookupLocation(sess); err != nil {
		r.emitter.EmitFailureGeneric(failureIPLookup, err.Error())
		r.emitter.EmitFailureGeneric(failureASNLookup, err.Error())
		r.emitter.EmitFailureGeneric(failureCCLookup, err.Error())
		r.emitter.EmitFailureGeneric(failureResolverLookup, err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.2, "geoip lookup")
	r.emitter.EmitStatusProgress(0.3, "resolver lookup")
	r.emitter.Emit(statusGeoIPLookup, eventStatusGeoIPLookup{
		ProbeIP:          sess.ProbeIP(),
		ProbeASN:         sess.ProbeASNString(),
		ProbeCC:          sess.ProbeCC(),
		ProbeNetworkName: sess.ProbeNetworkName(),
	})
	r.emitter.Emit(statusResolverLookup, eventStatusResolverLookup{
		ResolverASN:         sess.ResolverASNString(),
		ResolverIP:          sess.ResolverIP(),
		ResolverNetworkName: sess.ResolverNetworkName(),
	})

	builder.SetCallbacks(&runnerCallbacks{emitter: r.emitter})
	if len(r.settings.Inputs) <= 0 {
		switch builder.InputPolicy() {
		case engine.InputOrQueryBackend, engine.InputStrictlyRequired:
			r.emitter.EmitFailureStartup("no input provided")
			return
		}
		r.settings.Inputs = append(r.settings.Inputs, "")
	}
	experiment := builder.NewExperiment()
	defer func() {
		endEvent.DownloadedKB = experiment.KibiBytesReceived()
		endEvent.UploadedKB = experiment.KibiBytesSent()
	}()
	if !r.settings.Options.NoCollector {
		logger.Info("Opening report... please, be patient")
		if err := experiment.OpenReport(); err != nil {
			r.emitter.EmitFailureGeneric(failureReportCreate, err.Error())
			return
		}
		r.emitter.EmitStatusProgress(0.4, "open report")
		r.emitter.Emit(statusReportCreate, eventStatusReportGeneric{
			ReportID: experiment.ReportID(),
		})
	}
	// This deviates a little bit from measurement-kit, for which
	// a zero timeout is actually valid. Since it does not make much
	// sense, here we're changing the behaviour.
	//
	// See https://github.com/measurement-kit/measurement-kit/issues/1922
	if r.settings.Options.MaxRuntime > 0 {
		// We want to honour max_runtime only when we're running an
		// experiment that clearly wants specific input. We could refine
		// this policy in the future, but for now this covers in a
		// reasonable way web connectivity, so we should be ok.
		switch builder.InputPolicy() {
		case engine.InputOrQueryBackend, engine.InputStrictlyRequired:
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(
				ctx, time.Duration(r.settings.Options.MaxRuntime)*time.Second,
			)
			defer cancel()
		}
	}
	inputCount := len(r.settings.Inputs)
	start := time.Now()
	inflatedMaxRuntime := r.settings.Options.MaxRuntime + r.settings.Options.MaxRuntime/10
	eta := start.Add(time.Duration(inflatedMaxRuntime) * time.Second)
	for idx, input := range r.settings.Inputs {
		if ctx.Err() != nil {
			break
		}
		logger.Infof("Starting measurement with index %d", idx)
		r.emitter.Emit(statusMeasurementStart, eventMeasurementGeneric{
			Idx:   int64(idx),
			Input: input,
		})
		if input != "" && inputCount > 0 {
			var percentage float64
			if r.settings.Options.MaxRuntime > 0 {
				now := time.Now()
				percentage = (now.Sub(start).Seconds()/eta.Sub(start).Seconds())*0.6 + 0.4
			} else {
				percentage = (float64(idx)/float64(inputCount))*0.6 + 0.4
			}
			r.emitter.EmitStatusProgress(percentage, fmt.Sprintf(
				"processing %s", input,
			))
		}
		m, err := experiment.MeasureWithContext(
			r.contextForExperiment(ctx, builder),
			input,
		)
		if builder.Interruptible() && ctx.Err() != nil {
			// We want to stop here only if interruptible otherwise we want to
			// submit measurement and stop at beginning of next iteration
			break
		}
		m.AddAnnotations(r.settings.Annotations)
		if err != nil {
			r.emitter.Emit(failureMeasurement, eventMeasurementGeneric{
				Failure: err.Error(),
				Idx:     int64(idx),
				Input:   input,
			})
			// fallthrough: we want to submit the report anyway
		}
		data, err := json.Marshal(m)
		runtimex.PanicOnError(err, "measurement.MarshalJSON failed")
		r.emitter.Emit(measurement, eventMeasurementGeneric{
			Idx:     int64(idx),
			Input:   input,
			JSONStr: string(data),
		})
		if !r.settings.Options.NoCollector {
			logger.Info("Submitting measurement... please, be patient")
			err := experiment.SubmitAndUpdateMeasurement(m)
			r.emitter.Emit(measurementSubmissionEventName(err), eventMeasurementGeneric{
				Idx:     int64(idx),
				Input:   input,
				JSONStr: string(data),
				Failure: measurementSubmissionFailure(err),
			})
		}
		r.emitter.Emit(statusMeasurementDone, eventMeasurementGeneric{
			Idx:   int64(idx),
			Input: input,
		})
	}
}

func measurementSubmissionEventName(err error) string {
	if err != nil {
		return failureMeasurementSubmission
	}
	return statusMeasurementSubmission
}

func measurementSubmissionFailure(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
