package main

//
// Nettest task
//

import (
	"context"
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// newNettestRunner creates a new instance of nettestRunner.
func newNettestRunner() taskRunner {
	return &nettestRunner{}
}

// nettestRunner is the nettest taskRunner.
type nettestRunner struct{}

var _ taskRunner = &nettestRunner{}

// main implements taskRunner.main.
func (r *nettestRunner) main(ctx context.Context, emitter taskMaybeEmitter, args []byte) {
	logger := newTaskLogger(emitter)
	var config NettestConfig
	if err := json.Unmarshal(args, &config); err != nil {
		logger.Warnf("nettest: cannot parse settings: %s", err.Error())
		return
	}
	logger.verbose = config.Session.LogLevel == LogLevelDebug
	sess, err := newSession(ctx, &config.Session, logger)
	if err != nil {
		logger.Warnf("nettest: cannot create a new session: %s", err.Error())
		return
	}
	defer sess.Close()
	if err := sess.MaybeLookupBackendsContext(ctx); err != nil {
		logger.Warnf("nettest: cannot lookup backends: %s", err.Error())
		return
	}
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		logger.Warnf("nettest: cannot lookup location: %s", err.Error())
		return
	}
	reportFile := config.ReportFile
	if reportFile == "" {
		reportFile = "report.jsonl" // as documented
	}
	exp := &oonirun.Experiment{
		Annotations: config.Annotations,
		Callbacks: &nettestRunnerCallbacks{
			emitter: emitter,
		},
		ExtraOptions:   config.ExtraOptions,
		Inputs:         config.Inputs,
		InputFilePaths: config.InputFilePaths,
		MaxRuntime:     config.MaxRuntime,
		Name:           config.Name,
		NoCollector:    config.NoCollector,
		NoJSON:         config.NoJSON,
		Random:         config.Random,
		ReportFile:     reportFile,
		Session:        sess,
	}
	if err = exp.Run(ctx); err != nil {
		logger.Warnf("nettest: cannot run experiment: %s", err.Error())
		return
	}
}

// nettestRunnerCallbacks implements model.ExperimentCallbacks
type nettestRunnerCallbacks struct {
	// emitter is the underlying emitter to use.
	emitter taskMaybeEmitter
}

var _ model.ExperimentCallbacks = &nettestRunnerCallbacks{}

// OnProgress implements model.ExperimentCallbacks.OnProgress.
func (cb *nettestRunnerCallbacks) OnProgress(percentage float64, message string) {
	event := &ProgressEventValue{
		Percentage: percentage,
		Message:    message,
	}
	cb.emitter.maybeEmitEvent(ProgressEventName, event)
}

// OnData implements model.ExperimentCallbacks.OnData.
func (cb *nettestRunnerCallbacks) OnData(kibiBytesSent, kibiBytesReceived float64) {
	event := &DataUsageEventValue{
		KibiBytesSent:     kibiBytesSent,
		KibiBytesReceived: kibiBytesReceived,
	}
	cb.emitter.maybeEmitEvent(DataUsageEventName, event)
}

// OnMeasurementSubmission implements ExperimentCallbacks
func (cb *nettestRunnerCallbacks) OnMeasurementSubmission(idx int, m *model.Measurement, err error) {
	data, marshalErr := json.Marshal(m)
	runtimex.PanicOnError(marshalErr, "json.Marshal failed")
	event := &SubmitEventValue{
		Failure:     newFailureString(err),
		Index:       int64(idx),
		Input:       string(m.Input),
		ReportID:    m.ReportID,
		Measurement: string(data),
	}
	cb.emitter.maybeEmitEvent(SubmitEventName, event)
}
