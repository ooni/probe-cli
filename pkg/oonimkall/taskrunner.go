package oonimkall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// runnerForTask runs a specific task
type runnerForTask struct {
	emitter        *taskEmitterWrapper
	kvStoreBuilder taskKVStoreFSBuilder
	sessionBuilder taskSessionBuilder
	settings       *settings
}

var _ taskRunner = &runnerForTask{}

// newRunner creates a new task runner
func newRunner(settings *settings, emitter taskEmitter) *runnerForTask {
	return &runnerForTask{
		emitter:        &taskEmitterWrapper{emitter},
		kvStoreBuilder: &taskKVStoreFSBuilderEngine{},
		sessionBuilder: &taskSessionBuilderEngine{},
		settings:       settings,
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
	kvstore, err := r.kvStoreBuilder.NewFS(r.settings.StateDir)
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
		config.AvailableProbeServices = []model.Service{{
			Type:    "https",
			Address: r.settings.Options.ProbeServicesBaseURL,
		}}
	}
	return r.sessionBuilder.NewSession(ctx, config)
}

func (r *runnerForTask) contextForExperiment(
	ctx context.Context, builder taskExperimentBuilder,
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
func (r *runnerForTask) Run(ctx context.Context) {
	var logger model.Logger = newTaskLogger(r.emitter, r.settings.LogLevel)
	r.emitter.Emit(eventTypeStatusQueued, eventEmpty{})
	if r.hasUnsupportedSettings() {
		// event failureStartup already emitted
		return
	}
	r.emitter.Emit(eventTypeStatusStarted, eventEmpty{})
	sess, err := r.newsession(ctx, logger)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	endEvent := new(eventStatusEnd)
	defer func() {
		sess.Close()
		r.emitter.Emit(eventTypeStatusEnd, endEvent)
	}()

	builder, err := sess.NewExperimentBuilderByName(r.settings.Name)
	if err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}

	logger.Info("Looking up OONI backends... please, be patient")
	if err := sess.MaybeLookupBackendsContext(ctx); err != nil {
		r.emitter.EmitFailureStartup(err.Error())
		return
	}
	r.emitter.EmitStatusProgress(0.1, "contacted bouncer")

	logger.Info("Looking up your location... please, be patient")
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
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

	switch builder.InputPolicy() {
	case engine.InputOrQueryBackend, engine.InputStrictlyRequired:
		if len(r.settings.Inputs) <= 0 {
			r.emitter.EmitFailureStartup("no input provided")
			return
		}
	case engine.InputOrStaticDefault:
		if len(r.settings.Inputs) <= 0 {
			// TODO(bassosimone): before ooni/probe#1814 stunreachability
			// had a default input and was working on mobile. Now it doesn't
			// have any default input. We cannot break it. So, we're doing
			// something similar to what did before, i.e., we provide
			// input for this experiment. As a collaterall effect, we also
			// provide input for the dnscheck experiment. Yet, it kinda
			// sucks that a user would not see the URLs in the app. We
			// should do the following: (1) consolidate the code for running
			// experiments so that the code in here uses the same code we
			// are using for ooniprobe and miniooni; (2) change the app so
			// that we always query the backend or use default input from
			// here and the app reacts to our changes. If we do that, we
			// are easily able to show results in the apps.
			inputs, err := engine.StaticBareInputForExperiment(r.settings.Name)
			if err != nil {
				r.emitter.EmitFailureStartup("no input provided")
				return
			}
			r.settings.Inputs = inputs
		}
	case engine.InputOptional:
		// do nothing in this case
	default: // treat this case as engine.InputNone.
		if len(r.settings.Inputs) > 0 {
			r.emitter.EmitFailureStartup("experiment does not accept input")
			return
		}
		r.settings.Inputs = append(r.settings.Inputs, "")
	}
	experiment := builder.NewExperimentInstance()
	defer func() {
		endEvent.DownloadedKB = experiment.KibiBytesReceived()
		endEvent.UploadedKB = experiment.KibiBytesSent()
	}()
	if !r.settings.Options.NoCollector {
		logger.Info("Opening report... please, be patient")
		if err := experiment.OpenReportContext(ctx); err != nil {
			r.emitter.EmitFailureGeneric(eventTypeFailureReportCreate, err.Error())
			return
		}
		r.emitter.EmitStatusProgress(0.4, "open report")
		r.emitter.Emit(eventTypeStatusReportCreate, eventStatusReportGeneric{
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
		r.emitter.Emit(eventTypeStatusMeasurementStart, eventMeasurementGeneric{
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
			r.emitter.Emit(eventTypeFailureMeasurement, eventMeasurementGeneric{
				Failure: err.Error(),
				Idx:     int64(idx),
				Input:   input,
			})
			// Historical note: here we used to fallthrough but, since we have
			// implemented async measurements, the case where there is an error
			// and we also have a valid measurement cant't happen anymore. So,
			// now the only valid strategy here is to continue.
			continue
		}
		data, err := json.Marshal(m)
		runtimex.PanicOnError(err, "measurement.MarshalJSON failed")
		r.emitter.Emit(eventTypeMeasurement, eventMeasurementGeneric{
			Idx:     int64(idx),
			Input:   input,
			JSONStr: string(data),
		})
		if !r.settings.Options.NoCollector {
			logger.Info("Submitting measurement... please, be patient")
			err := experiment.SubmitAndUpdateMeasurementContext(ctx, m)
			r.emitter.Emit(measurementSubmissionEventName(err), eventMeasurementGeneric{
				Idx:     int64(idx),
				Input:   input,
				JSONStr: string(data),
				Failure: measurementSubmissionFailure(err),
			})
		}
		r.emitter.Emit(eventTypeStatusMeasurementDone, eventMeasurementGeneric{
			Idx:   int64(idx),
			Input: input,
		})
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
