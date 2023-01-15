package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/oonirunx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

var (
	// TODO: we should probably have groups.json as part of the default OONI
	// config in $OONIDir
	pathToGroups = "./internal/cmd/tinyooni/groups.json"

	//
	groups map[string]json.RawMessage
)

func registerRunGroup(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	subCmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a given experiment group",
		Args:  cobra.NoArgs,
	}
	rootCmd.AddCommand(subCmd)
	registerGroups(subCmd, globalOptions)
}

func registerGroups(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	data, err := os.ReadFile(pathToGroups)
	runtimex.PanicOnError(err, "registerGroups failed: could not read groups.json")

	err = json.Unmarshal(data, &groups)
	runtimex.PanicOnError(err, "json.Unmarshal failed")

	for name := range groups {
		subCmd := &cobra.Command{
			Use:   name,
			Short: fmt.Sprintf("Runs the %s experiment group", name),
			Run: func(cmd *cobra.Command, args []string) {
				runGroupMain(cmd.Use, globalOptions)
			},
		}
		rootCmd.AddCommand(subCmd)
	}
}

func runGroupMain(experimentName string, globalOptions *GlobalOptions) {
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
	var descr oonirunx.V2Descriptor
	err = json.Unmarshal(groups[experimentName], &descr)
	runtimex.PanicOnError(err, "json.Unmarshal failed")
	if err := oonirunx.V2MeasureDescriptor(ctx, cfg, &descr); err != nil {
		logger.Warnf("oonirun: running link failed: %s", err.Error())
	}
}
