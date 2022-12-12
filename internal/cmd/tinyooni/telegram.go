package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// telegramOptions contains options for web connectivity.
type telegramOptions struct {
	Annotations []string
}

func registerTelegram(rootCmd *cobra.Command, globalOptions *GlobalOptions) {
	options := &telegramOptions{}

	subCmd := &cobra.Command{
		Use:   "telegram",
		Short: "Runs the telegram experiment",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			telegramMain(globalOptions, options)
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
}

func telegramMain(globalOptions *GlobalOptions, options *telegramOptions) {
	ctx := context.Background()

	// create a new measurement session
	sess, err := newSession(ctx, globalOptions)
	runtimex.PanicOnError(err, "newSession failed")

	err = sess.MaybeLookupLocationContext(ctx)
	runtimex.PanicOnError(err, "sess.MaybeLookupLocation failed")

	db, err := database.Open("database.sqlite3")
	runtimex.PanicOnError(err, "database.Open failed")

	networkDB, err := db.CreateNetwork(sess)
	runtimex.PanicOnError(err, "db.Create failed")

	dbResult, err := db.CreateResult(".", "custom", networkDB.ID)
	runtimex.PanicOnError(err, "db.CreateResult failed")

	args := &model.ExperimentMainArgs{
		Annotations:    map[string]string{}, // TODO(bassosimone): fill
		CategoryCodes:  nil,                 // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db,
		Inputs:         nil,
		MaxRuntime:     0,
		MeasurementDir: "results.d",
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       dbResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}

	err = telegram.Main(ctx, args)
	runtimex.PanicOnError(err, "telegram.Main failed")
}
