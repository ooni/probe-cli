package oonimkall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// runnerForTask runs a specific task
type runnerForTask struct {
	emitter    *taskEmitterWrapper
	newKVStore func(path string) (model.KeyValueStore, error)
	newSession func(ctx context.Context, config engine.SessionConfig) (taskSession, error)
	settings   *settings
}

var _ taskRunner = &runnerForTask{}

// newRunner creates a new task runner
func newRunner(settings *settings, emitter taskEmitter) *runnerForTask {
	return &runnerForTask{
		emitter: &taskEmitterWrapper{emitter},
		newKVStore: func(path string) (model.KeyValueStore, error) {
			// Note that we will return a non-nil model.KeyValueStore even if the
			// kvstore.NewFS factory returns a nil *kvstore.FS because of how golang
			// converts between nil types. Because we're checking the error and
			// acting upon it, it is not a big deal.
			return kvstore.NewFS(path)
		},
		newSession: func(ctx context.Context, config engine.SessionConfig) (taskSession, error) {
			// Same note as above: the returned session is not nil even when the
			// factory returns a nil *engine.Session because of golang nil conversion.
			return engine.NewSession(ctx, config)
		},
		settings: settings,
	}
}

// failureInvalidVersion is the failure returned when Version is invalid
const failureInvalidVersion = "invalid Settings.Version number"

func (r *runnerForTask) hasUnsupportedSettings() bool {
	if r.settings.Version < taskABIVersion {
		r.emitter.EmitFailureStartup(failureInvalidVersion)
		return true
	}
	return false
}

func (r *runnerForTask) newsession(ctx context.Context, logger model.Logger) (taskSession, error) {
	kvstore, err := r.newKVStore(r.settings.StateDir)
	if err != nil {
		return nil, err
	}

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
		config.AvailableProbeServices = []model.OOAPIService{{
			Type:    "https",
			Address: r.settings.Options.ProbeServicesBaseURL,
		}}
	}
	return r.newSession(ctx, config)
}

// contextForExperiment ensurs that for measuring we only use an
// interruptible context when we can interrupt the experiment
func (r *runnerForTask) contextForExperiment(
	ctx context.Context, builder model.ExperimentBuilder,
) context.Context {
	if builder.Interruptible() {
		return ctx
	}
	return context.Background()
}

type runnerCallbacks struct {
	emitter taskEmitter
}

func (cb *runnerCallbacks) OnProgress(percentage float64, message string) {
	cb.emitter.Emit(eventTypeStatusProgress, eventStatusProgress{
		Percentage: 0.4 + (percentage * 0.6), // open report is 40%
		Message:    message,
	})
}

// Run runs the runner until completion. The context argument controls
// when to stop when processing multiple inputs, as well as when to stop
// experiments explicitly marked as interruptible.
func (r *runnerForTask) Run(rootCtx context.Context) {
	// Implementation note: this function uses these contexts:
	//
	// - rootCtx is the root context and is controlled by the user;
	//
	// - measCtx derives from rootCtx and is possibly tied to the
	// maximum runtime and is used to choose when to stop measuring;
	//
	// - submitCtx is like measCtx but, in case we're using a max
	// runtime, is given more time to finish submitting.
	//
	// See https://github.com/ooni/probe/issues/2037.
	var logger model.Logger = newTaskLogger(r.emitter, r.settings.LogLevel)
	r.emitter.Emit(eventTypeStatusQueued, eventEmpty{})
	if r.hasUnsupportedSettings() {
		// event failureStartup already emitted
		return
	}
	r.emitter.Emit(eventTypeStatusStarted, eventEmpty{})
	sess, err := r.newsession(rootCtx, logger)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	endEvent := new(eventStatusEnd)
	defer func() {
		_ = sess.Close()
		r.emitter.Emit(eventTypeStatusEnd, endEvent)
	}()

	builder, err := sess.NewExperimentBuilder(r.settings.Name)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	logger.Info("Looking up OONI backends... please, be patient")
	if err := sess.MaybeLookupBackendsContext(rootCtx); err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.1, "contacted bouncer")

	logger.Info("Looking up your location... please, be patient")
	if err := sess.MaybeLookupLocationContext(rootCtx); err != nil {
		r.emitter.EmitFailureGeneric(eventTypeFailureIPLookup, err.Error())
		r.emitter.EmitFailureGeneric(eventTypeFailureASNLookup, err.Error())
		r.emitter.EmitFailureGeneric(eventTypeFailureCCLookup, err.Error())
		r.emitter.EmitFailureGeneric(eventTypeFailureResolverLookup, err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.2, "geoip lookup")
	r.emitter.EmitStatusProgress(0.3, "resolver lookup")
	r.emitter.Emit(eventTypeStatusGeoIPLookup, eventStatusGeoIPLookup{
		ProbeIP:          sess.ProbeIP(),
		ProbeASN:         sess.ProbeASNString(),
		ProbeCC:          sess.ProbeCC(),
		ProbeNetworkName: sess.ProbeNetworkName(),
	})
	r.emitter.Emit(eventTypeStatusResolverLookup, eventStatusResolverLookup{
		ResolverASN:         sess.ResolverASNString(),
		ResolverIP:          sess.ResolverIP(),
		ResolverNetworkName: sess.ResolverNetworkName(),
	})

	builder.SetCallbacks(&runnerCallbacks{emitter: r.emitter})

	// Load targets. Note that, for Web Connectivity, the mobile app has
	// already loaded inputs and provides them as r.settings.Inputs.
	loader := builder.NewTargetLoader(&model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			// Not needed since the app already provides the
			// inputs to use for Web Connectivity.
		},
		Session:      sess,
		StaticInputs: r.settings.Inputs,
		SourceFiles:  []string{},
	})
	targets, err := loader.Load(rootCtx)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	experiment := builder.NewExperiment()
	defer func() {
		endEvent.DownloadedKB = experiment.KibiBytesReceived()
		endEvent.UploadedKB = experiment.KibiBytesSent()
	}()
	if !r.settings.Options.NoCollector {
		logger.Info("Opening report... please, be patient")
		if err := experiment.OpenReportContext(rootCtx); err != nil {
			r.emitter.EmitFailureGeneric(eventTypeFailureReportCreate, err.Error())
			return
		}
		r.emitter.EmitStatusProgress(0.4, "open report")
		r.emitter.Emit(eventTypeStatusReportCreate, eventStatusReportGeneric{
			ReportID: experiment.ReportID(),
		})
	}
	measCtx, measCancel := context.WithCancel(rootCtx)
	defer measCancel()
	submitCtx, submitCancel := context.WithCancel(rootCtx)
	defer submitCancel()

	// This deviates a little bit from measurement-kit, for which
	// a zero timeout is actually valid. Since it does not make much
	// sense, here we're changing the behaviour.
	//
	// See https://github.com/measurement-kit/measurement-kit/issues/1922
	if r.settings.Options.MaxRuntime > 0 && len(targets) > 1 {
		var (
			cancelMeas   context.CancelFunc
			cancelSubmit context.CancelFunc
		)
		// We give the context used for submitting extra time so that
		// it's possible to submit the last measurement.
		//
		// See https://github.com/ooni/probe/issues/2037 for more info.
		maxRuntime := time.Duration(r.settings.Options.MaxRuntime) * time.Second
		measCtx, cancelMeas = context.WithTimeout(measCtx, maxRuntime)
		defer cancelMeas()
		maxRuntime += 30 * time.Second
		submitCtx, cancelSubmit = context.WithTimeout(submitCtx, maxRuntime)
		defer cancelSubmit()
	}

	inputCount := len(targets)
	start := time.Now()
	inflatedMaxRuntime := r.settings.Options.MaxRuntime + r.settings.Options.MaxRuntime/10
	eta := start.Add(time.Duration(inflatedMaxRuntime) * time.Second)
	for idx, target := range targets {
		if measCtx.Err() != nil {
			break
		}
		logger.Infof("Starting measurement with index %d", idx)
		r.emitter.Emit(eventTypeStatusMeasurementStart, eventMeasurementGeneric{
			Idx:   int64(idx),
			Input: target.Input(),
		})
		if target.Input() != "" && inputCount > 0 {
			var percentage float64
			if r.settings.Options.MaxRuntime > 0 {
				now := time.Now()
				percentage = (now.Sub(start).Seconds()/eta.Sub(start).Seconds())*0.6 + 0.4
			} else {
				percentage = (float64(idx)/float64(inputCount))*0.6 + 0.4
			}
			r.emitter.EmitStatusProgress(percentage, fmt.Sprintf(
				"processing %s", target,
			))
		}

		// Richer input implementation note: in mobile, we only observe richer input
		// for Web Connectivity and only store this kind of input into the database and
		// otherwise we ignore richer input for other experiments, which are just
		// treated as experimental. As such, the thinking here is that we do not care
		// about *passing* richer input from desktop to mobile for some time. When
		// we will care, it would most likely suffice to require the Inputs field to
		// implement in Java the [model.ExperimentTarget] interface, which is something
		// we can always do, since it only has string accessors.
		m, err := experiment.MeasureWithContext(
			r.contextForExperiment(measCtx, builder),
			target,
		)

		if builder.Interruptible() && measCtx.Err() != nil {
			// We want to stop here only if interruptible otherwise we want to
			// submit measurement and stop at beginning of next iteration
			break
		}
		if err != nil {
			r.emitter.Emit(eventTypeFailureMeasurement, eventMeasurementGeneric{
				Failure: err.Error(),
				Idx:     int64(idx),
				Input:   target.Input(),
			})
			// Historical note: here we used to fallthrough but, since we have
			// implemented async measurements, the case where there is an error
			// and we also have a valid measurement cant't happen anymore. So,
			// now the only valid strategy here is to continue.
			continue
		}
		m.AddAnnotations(r.settings.Annotations)
		data, err := json.Marshal(m)
		runtimex.PanicOnError(err, "measurement.MarshalJSON failed")
		r.emitter.Emit(eventTypeMeasurement, eventMeasurementGeneric{
			Idx:     int64(idx),
			Input:   target.Input(),
			JSONStr: string(data),
		})
		if !r.settings.Options.NoCollector {
			logger.Info("Submitting measurement... please, be patient")
			err := experiment.SubmitAndUpdateMeasurementContext(submitCtx, m)
			warnOnFailure(logger, "cannot submit measurement", err)
			r.emitter.Emit(measurementSubmissionEventName(err), eventMeasurementGeneric{
				Idx:     int64(idx),
				Input:   target.Input(),
				JSONStr: string(data),
				Failure: measurementSubmissionFailure(err),
			})
		}
		r.emitter.Emit(eventTypeStatusMeasurementDone, eventMeasurementGeneric{
			Idx:   int64(idx),
			Input: target.Input(),
		})
	}
}

func warnOnFailure(logger model.Logger, message string, err error) {
	if err != nil {
		logger.Warnf("%s: %s (%+v)", message, err.Error(), err)
	}
}

func measurementSubmissionEventName(err error) string {
	if err != nil {
		return eventTypeFailureMeasurementSubmission
	}
	return eventTypeStatusMeasurementSubmission
}

func measurementSubmissionFailure(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
