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
	// - loadCtx derives from rootCtx and is used to load inputs;
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

	// check whether we support the provided settings
	if r.hasUnsupportedSettings() {
		// event failureStartup already emitted
		return
	}
	r.emitter.Emit(eventTypeStatusStarted, eventEmpty{})

	// create a new measurement session
	sess, err := r.newsession(rootCtx, logger)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	// make sure we emit the status.end event when we're done
	endEvent := new(eventStatusEnd)
	defer func() {
		_ = sess.Close()
		r.emitter.Emit(eventTypeStatusEnd, endEvent)
	}()

	// create an experiment builder for the given experiment name
	builder, err := sess.NewExperimentBuilder(r.settings.Name)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	// choose the proper OONI backend to use
	logger.Info("Looking up OONI backends... please, be patient")
	if err := sess.MaybeLookupBackendsContext(rootCtx); err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.1, "contacted bouncer")

	// discover the probe location
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

	// configure the callbacks to emit events
	builder.SetCallbacks(&runnerCallbacks{emitter: r.emitter})

	// load targets using the experiment-specific loader
	loader := builder.NewTargetLoader(&model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			// TODO(https://github.com/ooni/probe/issues/2766): to correctly load Web Connectivity targets
			// here we need to honour the relevant check-in settings.
		},
		Session:      sess,
		StaticInputs: r.settings.Inputs,
		SourceFiles:  []string{},
	})
	loadCtx, loadCancel := context.WithTimeout(rootCtx, 30*time.Second)
	defer loadCancel()
	targets, err := loader.Load(loadCtx)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	// create the new experiment
	experiment := builder.NewExperiment()

	// make sure we account for the bytes sent and received
	defer func() {
		endEvent.DownloadedKB = experiment.KibiBytesReceived()
		endEvent.UploadedKB = experiment.KibiBytesSent()
	}()

	// open a new report if possible
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

	// create the default context for measuring
	measCtx, measCancel := context.WithCancel(rootCtx)
	defer measCancel()

	// create the default context for submitting
	submitCtx, submitCancel := context.WithCancel(rootCtx)
	defer submitCancel()

	// Update measCtx and submitCtx to be timeout bound in case there's
	// more than one input/target to measure.
	//
	// This deviates a little bit from measurement-kit, for which
	// a zero timeout is actually valid. Since it does not make much
	// sense, here we're changing the behaviour.
	//
	// Additionally, since https://github.com/ooni/probe-cli/pull/1620,
	// we honour the MaxRuntime for all experiments that have more
	// than one input. Previously, it was just Web Connectivity, yet,
	// it seems reasonable to honour MaxRuntime everytime the whole
	// experiment runtime depends on more than one input.
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

	// prepare for cycling through the targets
	inputCount := len(targets)
	start := time.Now()
	inflatedMaxRuntime := r.settings.Options.MaxRuntime + r.settings.Options.MaxRuntime/10
	eta := start.Add(time.Duration(inflatedMaxRuntime) * time.Second)

	for idx, target := range targets {
		// handle the case where the time allocated for measuring has elapsed
		if measCtx.Err() != nil {
			break
		}

		// notify the mobile app that we are about to measure a specific target
		//
		// note that here we provide also the CategoryCode and the CountryCode
		// so that the mobile app can update its URLs table here
		logger.Infof("Starting measurement with index %d", idx)
		r.emitter.Emit(eventTypeStatusMeasurementStart, eventMeasurementGeneric{
			CategoryCode: target.Category(),
			CountryCode:  target.Country(),
			Idx:          int64(idx),
			Input:        target.Input(),
		})

		// emit progress when there is more than one target to measure
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

		// Perform the measurement proper.
		m, err := experiment.MeasureWithContext(
			r.contextForExperiment(measCtx, builder),
			target,
		)

		// Handle the case where our time for measuring has elapsed while
		// we were measuring and assume the context interrupted the measurement
		// midway, so it doesn't make sense to submit it.
		if builder.Interruptible() && measCtx.Err() != nil {
			break
		}

		// handle the case where the measurement has failed
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

		// make sure the measurement contains the user-specified annotations
		m.AddAnnotations(r.settings.Annotations)

		// serialize the measurement to JSON (cannot fail in practice)
		data, err := json.Marshal(m)
		runtimex.PanicOnError(err, "measurement.MarshalJSON failed")

		// let the mobile app know about this measurement
		r.emitter.Emit(eventTypeMeasurement, eventMeasurementGeneric{
			Idx:     int64(idx),
			Input:   target.Input(),
			JSONStr: string(data),
		})

		// if possible, submit the measurement to the OONI backend
		if !r.settings.Options.NoCollector {
			logger.Info("Submitting measurement... please, be patient")
			muid, err := experiment.SubmitAndUpdateMeasurementContext(submitCtx, m)
			warnOnFailure(logger, "cannot submit measurement", err)
			r.emitter.Emit(measurementSubmissionEventName(err), eventMeasurementGeneric{
				Idx:            int64(idx),
				Input:          target.Input(),
				JSONStr:        string(data),
				Failure:        measurementSubmissionFailure(err),
				MeasurementUID: muid,
			})
		}

		// let the app know that we're done measuring this entry
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
