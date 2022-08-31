package main

//
// Run eXperiment by name
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// runx runs the given experiment by name
func runx(ctx context.Context, sess oonirun.Session, experimentName string,
	annotations map[string]string, extraOptions map[string]any, currentOptions Options) {
	desc := &oonirun.Experiment{
		Annotations:    annotations,
		ExtraOptions:   extraOptions,
		Inputs:         currentOptions.Inputs,
		InputFilePaths: currentOptions.InputFilePaths,
		MaxRuntime:     currentOptions.MaxRuntime,
		Name:           experimentName,
		NoCollector:    currentOptions.NoCollector,
		NoJSON:         currentOptions.NoJSON,
		Random:         currentOptions.Random,
		ReportFile:     currentOptions.ReportFile,
		Session:        sess,
	}
	err := desc.Run(ctx)
	runtimex.PanicOnError(err, "cannot run experiment")
}
