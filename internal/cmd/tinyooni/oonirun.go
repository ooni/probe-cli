package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/ooni/probe-cli/v3/internal/oonirun"
	"github.com/ooni/probe-cli/v3/internal/oonirunx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

type oonirunOptions struct {
	Inputs         []string
	InputFilePaths []string
}

func registerOoniRun(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	options := &oonirunOptions{}

	subCmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a given experiment group",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			ooniRunMain(options, globalOptions)
		},
	}
	rootCmd.AddCommand(subCmd)
	flags := subCmd.Flags()

	flags.StringSliceVarP(
		&options.Inputs,
		"input",
		"i",
		[]string{},
		"URL of the OONI Run v2 descriptor to run (may be specified multiple times)",
	)
	flags.StringSliceVarP(
		&options.InputFilePaths,
		"input-file",
		"f",
		[]string{},
		"Path to the OONI Run v2 descriptor to run (may be specified multiple times)",
	)
}

func ooniRunMain(options *oonirunOptions, globalOptions *GlobalOptions) {
	ctx := context.Background()

	// create a new measurement session
	sess, err := newSession(ctx, globalOptions)
	runtimex.PanicOnError(err, "newSession failed")

	err = sess.MaybeLookupLocationContext(ctx)
	runtimex.PanicOnError(err, "sess.MaybeLookupLocation failed")

	// initialize database
	dbProps := initDatabase(ctx, sess, globalOptions)

	logger := sess.Logger()
	cfg := &oonirunx.LinkConfig{
		AcceptChanges: globalOptions.Yes,
		KVStore:       sess.KeyValueStore(),
		NoCollector:   globalOptions.NoCollector,
		NoJSON:        globalOptions.NoJSON,
		ReportFile:    globalOptions.ReportFile,
		Session:       sess,
		DatabaseProps: dbProps,
	}
	for _, URL := range options.Inputs {
		r := oonirunx.NewLinkRunner(cfg, URL)
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
	for _, filename := range options.InputFilePaths {
		data, err := os.ReadFile(filename)
		if err != nil {
			logger.Warnf("oonirun: reading OONI Run v2 descriptor failed: %s", err.Error())
			continue
		}
		var descr oonirunx.V2Descriptor
		if err := json.Unmarshal(data, &descr); err != nil {
			logger.Warnf("oonirun: parsing OONI Run v2 descriptor failed: %s", err.Error())
			continue
		}
		if err := oonirunx.V2MeasureDescriptor(ctx, cfg, &descr); err != nil {
			logger.Warnf("oonirun: running link failed: %s", err.Error())
			continue
		}
	}
}
