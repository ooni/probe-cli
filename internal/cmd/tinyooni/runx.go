package main

import (
	"context"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/registryx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

func registerRunExperiment(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	subCmd := &cobra.Command{
		Use:   "runx",
		Short: "Runs a given experiment",
		Args:  cobra.NoArgs,
	}
	rootCmd.AddCommand(subCmd)
	registerAllExperiments(subCmd, globalOptions)
}

func registerAllExperiments(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	for name, factory := range registryx.AllExperiments {
		subCmd := &cobra.Command{
			Use:   name,
			Short: fmt.Sprintf("Runs the %s experiment", name),
			Args:  cobra.NoArgs,
			Run: func(cmd *cobra.Command, args []string) {
				runExperimentsMain(cmd.Use, globalOptions)
			},
		}
		rootCmd.AddCommand(subCmd)

		// build experiment specific flags here
		options := registryx.AllExperimentOptions[subCmd.Use]
		options.BuildFlags(subCmd.Use, subCmd)
		factory.SetOptions(options)
	}
}

func runExperimentsMain(experimentName string, currentOptions *GlobalOptions) {
	ctx := context.Background()

	// create a new measurement session
	sess, err := newSession(ctx, currentOptions)
	runtimex.PanicOnError(err, "newSession failed")

	err = sess.MaybeLookupLocationContext(ctx)
	runtimex.PanicOnError(err, "sess.MaybeLookupLocation failed")

	// initialize database
	dbProps := initDatabase(ctx, sess, currentOptions)

	factory := registryx.AllExperiments[experimentName]
	factory.SetArguments(sess, dbProps, nil)
	err = factory.Main(ctx)
	runtimex.PanicOnError(err, fmt.Sprintf("%s.Main failed", experimentName))
}
