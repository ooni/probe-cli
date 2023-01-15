package registryx

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

// webConnectivityOptions contains options for web connectivity.
type webConnectivityOptions struct {
	Annotations    []string
	InputFilePaths []string
	Inputs         []string
	MaxRuntime     int64
	Random         bool
	ConfigOptions  []string
}

func init() {
	options := &webConnectivityOptions{}
	AllExperiments["web_connectivity"] = &Factory{
		Main: func(ctx context.Context, sess *engine.Session, db *database.DatabaseProps) error {
			config := &webconnectivity.Config{}
			configMap := mustMakeMapStringAny(options.ConfigOptions)
			if err := setOptionsAny(config, configMap); err != nil {
				return err
			}
			return webconnectivityMain(ctx, sess, options, config, db)
		},
		Oonirun: func(ctx context.Context, sess *engine.Session, inputs []string,
			args map[string]any, extraOptions map[string]any, db *database.DatabaseProps) error {
			options := &webConnectivityOptions{}
			options.Inputs = inputs
			if err := setOptionsAny(options, args); err != nil {
				return err
			}
			config := &webconnectivity.Config{}
			if err := setOptionsAny(config, extraOptions); err != nil {
				return err
			}
			return webconnectivityMain(ctx, sess, options, config, db)
		},
		BuildFlags: func(experimentName string, rootCmd *cobra.Command) {
			webconnectivityBuildFlags(experimentName, rootCmd, options, &webconnectivity.Config{})
		},
	}
}

func webconnectivityMain(ctx context.Context, sess model.ExperimentSession, options *webConnectivityOptions,
	config *webconnectivity.Config, db *database.DatabaseProps) error {
	args := &model.ExperimentMainArgs{
		Annotations:    map[string]string{}, // TODO(bassosimone): fill
		CategoryCodes:  nil,                 // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db.Database,
		Inputs:         options.Inputs,
		MaxRuntime:     options.MaxRuntime,
		MeasurementDir: db.DatabaseResult.MeasurementDir,
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       db.DatabaseResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}

	return webconnectivity.Main(ctx, args, config)
}

func webconnectivityBuildFlags(experimentName string, rootCmd *cobra.Command,
	options *webConnectivityOptions, config any) {
	flags := rootCmd.Flags()

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
