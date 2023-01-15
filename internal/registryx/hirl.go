package registryx

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/hirl"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

type hirlOptions struct {
	Annotations   []string
	ConfigOptions []string
}

func init() {
	options := &hirlOptions{}
	AllExperiments["hirl"] = &Factory{
		Main: func(ctx context.Context, sess *engine.Session, db *database.DatabaseProps) error {
			config := &hirl.Config{}
			configMap := mustMakeMapStringAny(options.ConfigOptions)
			if err := setOptionsAny(config, configMap); err != nil {
				return err
			}
			return hirlMain(ctx, sess, options, config, db)
		},
		Oonirun: func(ctx context.Context, sess *engine.Session, inputs []string,
			args map[string]any, extraOptions map[string]any, db *database.DatabaseProps) error {
			options := &hirlOptions{}
			if err := setOptionsAny(options, args); err != nil {
				return err
			}
			config := &hirl.Config{}
			if err := setOptionsAny(config, extraOptions); err != nil {
				return err
			}
			return hirlMain(ctx, sess, options, config, db)
		},
		BuildFlags: func(experimentName string, rootCmd *cobra.Command) {
			hirlBuildFlags(experimentName, rootCmd, options, &hirl.Config{})
		},
	}
}

func hirlMain(ctx context.Context, sess model.ExperimentSession, options *hirlOptions,
	config *hirl.Config, db *database.DatabaseProps) error {
	args := &model.ExperimentMainArgs{
		Annotations:    map[string]string{}, // TODO(bassosimone): fill
		CategoryCodes:  nil,                 // accept any category
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
	return hirl.Main(ctx, args, config)
}

func hirlBuildFlags(experimentName string, rootCmd *cobra.Command,
	options *hirlOptions, config any) {
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
