package registryx

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

type riseupvpnOptions struct {
	Annotations   []string
	ConfigOptions []string
}

func init() {
	options := &riseupvpnOptions{}
	AllExperiments["riseupvpn"] = &Factory{
		Main: func(ctx context.Context, sess *engine.Session, db *database.DatabaseProps) error {
			config := &riseupvpn.Config{}
			configMap := mustMakeMapStringAny(options.ConfigOptions)
			if err := setOptionsAny(config, configMap); err != nil {
				return err
			}
			return riseupvpnMain(ctx, sess, options, config, db)
		},
		Oonirun: func(ctx context.Context, sess *engine.Session, inputs []string,
			args map[string]any, extraOptions map[string]any, db *database.DatabaseProps) error {
			options := &riseupvpnOptions{}
			if err := setOptionsAny(options, args); err != nil {
				return err
			}
			config := &riseupvpn.Config{}
			if err := setOptionsAny(config, extraOptions); err != nil {
				return err
			}
			return riseupvpnMain(ctx, sess, options, config, db)
		},
		BuildFlags: func(experimentName string, rootCmd *cobra.Command) {
			riseupvpnBuildFlags(experimentName, rootCmd, options, &riseupvpn.Config{})
		},
	}
}

func riseupvpnMain(ctx context.Context, sess model.ExperimentSession, options *riseupvpnOptions,
	config *riseupvpn.Config, db *database.DatabaseProps) error {
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
	return riseupvpn.Main(ctx, args, config)
}

func riseupvpnBuildFlags(experimentName string, rootCmd *cobra.Command,
	options *riseupvpnOptions, config any) {
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
