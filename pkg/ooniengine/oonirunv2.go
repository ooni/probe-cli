package main

//
// OONIRunV2 tasks
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
	"google.golang.org/protobuf/proto"
)

func init() {
	taskRegistry["OONIRunV2MeasureDescriptor"] = newOONIRunV2MeasureDescriptorRunner()
}

// newOONIRunV2MeasureDescriptorRunner creates a new oonirRunV2MeasureDescriptorRunner.
func newOONIRunV2MeasureDescriptorRunner() taskRunner {
	return &oonirRunV2MeasureDescriptorRunner{}
}

// oonirRunV2MeasureDescriptorRunner is the geoip task runner.
type oonirRunV2MeasureDescriptorRunner struct{}

var _ taskRunner = &oonirRunV2MeasureDescriptorRunner{}

// main implements taskRunner.main.
func (r *oonirRunV2MeasureDescriptorRunner) main(
	ctx context.Context, emitter taskMaybeEmitter, args []byte) {
	logger := newTaskLogger(emitter)
	var config abi.OONIRunV2MeasureDescriptorConfig
	if err := proto.Unmarshal(args, &config); err != nil {
		logger.Warnf("oonirunv2: cannot parse settings: %s", err.Error())
		return
	}
	// 🔥🔥🔥 Rule of thumb when reviewing protobuf code: if the code is using
	// the safe GetXXX accessors, it's good, otherwise it's not good
	logger.verbose = config.GetSession().GetLogLevel() == abi.LogLevel_DEBUG
	sess, err := newSession(ctx, config.GetSession(), logger)
	if err != nil {
		logger.Warnf("oonirunv2: cannot create a new session: %s", err.Error())
		return
	}
	defer sess.Close()
	if err := sess.MaybeLookupBackendsContext(ctx); err != nil {
		logger.Warnf("oonirunv2: cannot lookup backends: %s", err.Error())
		return
	}
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		logger.Warnf("oonirunv2: cannot lookup location: %s", err.Error())
		return
	}
	reportFile := config.GetReportFile()
	if reportFile == "" {
		reportFile = "report.jsonl" // as documented
	}
	cfg := &oonirun.LinkConfig{
		AcceptChanges: false,
		Callbacks: &nettestRunnerCallbacks{
			emitter: emitter,
		},
		KVStore:     sess.KeyValueStore(),
		MaxRuntime:  config.GetMaxRuntime(),
		NoCollector: config.GetNoCollector(),
		NoJSON:      config.GetNoJson(),
		Random:      config.GetRandom(),
		ReportFile:  reportFile,
		Session:     sess,
	}
	descriptor := &oonirun.V2Descriptor{
		Name:        config.GetV2Descriptor().GetName(),
		Description: config.GetV2Descriptor().GetDescription(),
		Author:      config.GetV2Descriptor().GetAuthor(),
		Nettests:    []oonirun.V2Nettest{},
	}
	for _, nettest := range config.GetV2Descriptor().GetNettests() {
		descriptor.Nettests = append(descriptor.Nettests, oonirun.V2Nettest{
			Annotations: nettest.GetAnnotations(),
			Inputs:      nettest.GetInputs(),
			Options:     parseSerializedOptions(nettest.GetOptions()),
			TestName:    nettest.GetTestName(),
		})
	}
	if err := oonirun.V2MeasureDescriptor(ctx, cfg, descriptor); err != nil {
		logger.Warnf("oonirunv2: cannot run descriptor: %s", err.Error())
		return
	}
}
