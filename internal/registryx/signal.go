package registryx

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

type signalOptions struct {
	Annotations   []string
	ConfigOptions []string
}

func init() {
	options := &signalOptions{}
	AllExperiments["signal"] = &Factory{
		Main: func(ctx context.Context, sess *engine.Session, db *database.DatabaseProps) error {
			config := &signal.Config{}
			configMap := mustMakeMapStringAny(options.ConfigOptions)
			if err := setOptionsAny(config, configMap); err != nil {
				return err
			}
			return signalMain(ctx, sess, options, config, db)
		},
		Oonirun: func(ctx context.Context, sess *engine.Session, inputs []string,
			args map[string]any, extraOptions map[string]any, db *database.DatabaseProps) error {
			options := &signalOptions{}
			if err := setOptionsAny(options, args); err != nil {
				return err
			}
			config := &signal.Config{}
			if err := setOptionsAny(config, extraOptions); err != nil {
				return err
			}
			return signalMain(ctx, sess, options, config, db)
		},
		BuildFlags: func(experimentName string, rootCmd *cobra.Command) {
			signalBuildFlags(experimentName, rootCmd, options, &signal.Config{})
		},
	}
}

func signalMain(ctx context.Context, sess model.ExperimentSession, options *signalOptions,
	config *signal.Config, db *database.DatabaseProps) error {
	annotations := mustMakeMapStringString(options.Annotations)
	args := &model.ExperimentMainArgs{
		Annotations:    annotations, // TODO(bassosimone): fill
		CategoryCodes:  nil,         // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db.Database,
		Inputs:         nil,
		MaxRuntime:     0,
		MeasurementDir: db.DatabaseResult.MeasurementDir,
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       db.DatabaseResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}
	return signal.Main(ctx, args, config)
}

func signalBuildFlags(experimentName string, rootCmd *cobra.Command,
	options *signalOptions, config any) {
	flags := rootCmd.Flags()

	flags.StringSliceVarP(
		&options.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := documentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&options.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
