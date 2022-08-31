package main

//
// OONI Run
//

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ooniRunMain runs the experiments described by the given OONI Run URLs. This
// function works with both v1 and v2 OONI Run URLs.
func ooniRunMain(ctx context.Context,
	sess *engine.Session, currentOptions Options, annotations map[string]string) {
	runtimex.PanicIfTrue(
		len(currentOptions.Inputs) <= 0,
		"in oonirun mode you need to specify at least one URL using `-i URL`",
	)
	runtimex.PanicIfTrue(
		len(currentOptions.InputFilePaths) > 0,
		"in oonirun mode you cannot specify any `-f FILE` file",
	)
	logger := sess.Logger()
	cfg := &oonirun.LinkConfig{
		AcceptChanges: currentOptions.Yes,
		Annotations:   annotations,
		KVStore:       sess.KeyValueStore(),
		MaxRuntime:    currentOptions.MaxRuntime,
		NoCollector:   currentOptions.NoCollector,
		NoJSON:        currentOptions.NoJSON,
		Random:        currentOptions.Random,
		ReportFile:    currentOptions.ReportFile,
		Session:       sess,
	}
	for _, URL := range currentOptions.Inputs {
		r := oonirun.NewLinkRunner(cfg, URL)
		if err := r.Run(ctx); err != nil {
			if errors.Is(err, oonirun.ErrNeedToAcceptChanges) {
				logger.Warnf("oonirun: to accept these changes, rerun adding `-y` to the command line")
				logger.Warnf("oonirun: we'll show this error every time the upstream link changes")
				panic("oonirun: need to accept changes using `-y`")
			}
			logger.Warnf("oonirun: running link failed: %s", err.Error())
			continue
		}
	}
}
