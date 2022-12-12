package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// webConnectivityOptions contains options for web connectivity.
type webConnectivityOptions struct {
	Annotations    []string
	InputFilePaths []string
	Inputs         []string
	MaxRuntime     int64
	Random         bool
}

func registerWebConnectivity(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	options := &webConnectivityOptions{}

	subCmd := &cobra.Command{
		Use:   "web_connectivity",
		Short: "Runs the webconnectivity experiment",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			webConnectivityMain(globalOptions, options)
		},
		Aliases: []string{"webconnectivity"},
	}
	rootCmd.AddCommand(subCmd)
	flags := subCmd.Flags()

	flags.StringSliceVarP(
		&options.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	flags.StringSliceVarP(
		&options.InputFilePaths,
		"input-file",
		"f",
		[]string{},
		"path to file to supply test dependent input (may be specified multiple times)",
	)

	flags.StringSliceVarP(
		&options.Inputs,
		"input",
		"i",
		[]string{},
		"add test-dependent input (may be specified multiple times)",
	)

	flags.Int64Var(
		&options.MaxRuntime,
		"max-runtime",
		0,
		"maximum runtime in seconds for the experiment (zero means infinite)",
	)

	flags.BoolVar(
		&options.Random,
		"random",
		false,
		"randomize the inputs list",
	)
}

func webConnectivityMain(globalOptions *GlobalOptions, options *webConnectivityOptions) {
	ctx := context.Background()

	ooniHome := maybeGetOONIDir(globalOptions.HomeDir)

	// create a new measurement session
	sess, err := newSession(ctx, globalOptions)
	runtimex.PanicOnError(err, "newSession failed")

	err = sess.MaybeLookupLocationContext(ctx)
	runtimex.PanicOnError(err, "sess.MaybeLookupLocation failed")

	db, err := database.Open(databasePath(ooniHome))
	runtimex.PanicOnError(err, "database.Open failed")

	networkDB, err := db.CreateNetwork(sess)
	runtimex.PanicOnError(err, "db.Create failed")

	dbResult, err := db.CreateResult(ooniHome, "custom", networkDB.ID)
	runtimex.PanicOnError(err, "db.CreateResult failed")

	args := &model.ExperimentMainArgs{
		Annotations:    map[string]string{}, // TODO(bassosimone): fill
		CategoryCodes:  nil,                 // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db,
		Inputs:         options.Inputs,
		MaxRuntime:     options.MaxRuntime,
		MeasurementDir: dbResult.MeasurementDir,
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       dbResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}

	err = webconnectivity.Main(ctx, args)
	runtimex.PanicOnError(err, "webconnectivity.Main failed")
}
