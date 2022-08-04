package main

//
// OONIRunV2 tasks
//

import (
	"context"
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/oonirun"
)

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
	var config OONIRunV2MeasureDescriptorConfig
	if err := json.Unmarshal(args, &config); err != nil {
		logger.Warnf("geoip: cannot parse settings: %s", err.Error())
		return
	}
	logger.verbose = config.Session.LogLevel == LogLevelDebug
	sess, err := newSession(ctx, &config.Session, logger)
	if err != nil {
		logger.Warnf("geoip: cannot create a new session: %s", err.Error())
		return
	}
	defer sess.Close()
	if err := sess.MaybeLookupBackendsContext(ctx); err != nil {
		logger.Warnf("nettest: cannot lookup backends: %s", err.Error())
		return
	}
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		logger.Warnf("geoip: cannot lookup location: %s", err.Error())
		return
	}
	reportFile := config.ReportFile
	if reportFile == "" {
		reportFile = "report.jsonl" // as documented
	}
	cfg := &oonirun.LinkConfig{
		AcceptChanges: false,
		Callbacks: &nettestRunnerCallbacks{
			emitter: emitter,
		},
		KVStore:     sess.KeyValueStore(),
		MaxRuntime:  config.MaxRuntime,
		NoCollector: config.NoCollector,
		NoJSON:      config.NoJSON,
		Random:      config.Random,
		ReportFile:  reportFile,
		Session:     sess,
	}
	descriptor := &oonirun.V2Descriptor{
		Name:        config.Descriptor.Name,
		Description: config.Descriptor.Description,
		Author:      config.Descriptor.Author,
		Nettests:    []oonirun.V2Nettest{},
	}
	for _, nettest := range config.Descriptor.Nettests {
		descriptor.Nettests = append(descriptor.Nettests, oonirun.V2Nettest{
			Annotations: nettest.Annotations,
			Inputs:      nettest.Inputs,
			Options:     nettest.Options,
			TestName:    nettest.TestName,
		})
	}
	if err := oonirun.V2MeasureDescriptor(ctx, cfg, descriptor); err != nil {
		logger.Warnf("oonirunv2: cannot run descriptor: %s", err.Error())
		return
	}
}
