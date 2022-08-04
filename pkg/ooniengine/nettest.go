package main

//
// Nettest task
//

import (
	"context"
	"encoding/json"
	"log"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
	"google.golang.org/protobuf/proto"
)

func init() {
	taskRegistry["Nettest"] = newNettestRunner()
}

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
	var config abi.NettestConfig
	if err := proto.Unmarshal(args, &config); err != nil {
		logger.Warnf("nettest: cannot parse settings: %s", err.Error())
		return
	}
	// 🔥🔥🔥 Rule of thumb when reviewing protobuf code: if the code is using
	// the safe GetXXX accessors, it's good, otherwise it's not good
	logger.verbose = config.GetSession().GetLogLevel() == abi.LogLevel_DEBUG
	sess, err := newSession(ctx, config.GetSession(), logger)
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
	reportFile := config.GetReportFile()
	if reportFile == "" {
		reportFile = "report.jsonl" // as documented
	}
	exp := &oonirun.Experiment{
		Annotations: config.GetAnnotations(),
		Callbacks: &nettestRunnerCallbacks{
			emitter: emitter,
		},
		ExtraOptions:   parseSerializedOptions(config.GetExtraOptions()),
		Inputs:         config.GetInputs(),
		InputFilePaths: config.GetInputFilePaths(),
		MaxRuntime:     config.GetMaxRuntime(),
		Name:           config.GetName(),
		NoCollector:    config.GetNoCollector(),
		NoJSON:         config.GetNoJson(),
		Random:         config.GetRandom(),
		ReportFile:     reportFile,
		Session:        sess,
	}
	if err = exp.Run(ctx); err != nil {
		logger.Warnf("nettest: cannot run experiment: %s", err.Error())
		return
	}
}

// parseSerializedOptions parses JSON serialized options to
// the correct map type. We cannot easily serialized a map
// using any keys with protobuf3, so we use JSON.
func parseSerializedOptions(data string) map[string]any {
	out := make(map[string]any)
	if err := json.Unmarshal([]byte(data), &out); err != nil {
		log.Printf("parseSerializedOptions: %s", err.Error())
		return map[string]any{}
	}
	return out
}

// nettestRunnerCallbacks implements model.ExperimentCallbacks
type nettestRunnerCallbacks struct {
	// emitter is the underlying emitter to use.
	emitter taskMaybeEmitter
}

var _ model.ExperimentCallbacks = &nettestRunnerCallbacks{}

// OnProgress implements model.ExperimentCallbacks.OnProgress.
func (cb *nettestRunnerCallbacks) OnProgress(percentage float64, message string) {
	event := &abi.ProgressEvent{
		Percentage: percentage,
		Message:    message,
	}
	cb.emitter.maybeEmitEvent("Progress", event)
}

// OnData implements model.ExperimentCallbacks.OnData.
func (cb *nettestRunnerCallbacks) OnData(kibiBytesSent, kibiBytesReceived float64) {
	event := &abi.DataUsageEvent{
		KibiBytesSent:     kibiBytesSent,
		KibiBytesReceived: kibiBytesReceived,
	}
	cb.emitter.maybeEmitEvent("DataUsage", event)
}

// OnMeasurementSubmission implements ExperimentCallbacks
func (cb *nettestRunnerCallbacks) OnMeasurementSubmission(idx int, m *model.Measurement, err error) {
	data, marshalErr := json.Marshal(m)
	runtimex.PanicOnError(marshalErr, "json.Marshal failed")
	event := &abi.SubmitEvent{
		Failure:     newFailureString(err),
		Index:       int64(idx),
		Input:       string(m.Input),
		ReportId:    m.ReportID,
		Measurement: string(data),
	}
	cb.emitter.maybeEmitEvent("Submit", event)
}
