package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
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

var _ model.ExperimentOptions = &webConnectivityOptions{}

func init() {
	options := &webConnectivityOptions{}
	AllExperimentOptions["web_connectivity"] = options
	AllExperiments["web_connectivity"] = &webconnectivity.ExperimentMain{}
}

func (wco *webConnectivityOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	return &model.ExperimentMainArgs{
		Annotations:    map[string]string{}, // TODO(bassosimone): fill
		CategoryCodes:  nil,                 // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db.Database,
		Inputs:         wco.Inputs,
		MaxRuntime:     wco.MaxRuntime,
		MeasurementDir: db.DatabaseResult.MeasurementDir,
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       db.DatabaseResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}
}

func (wco *webConnectivityOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(wco.ConfigOptions)
}

func (wco *webConnectivityOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	wco.Inputs = inputs
	if err := setter.SetOptionsAny(wco, args); err != nil {
		return err
	}
	return nil
}

func (wco *webConnectivityOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &webconnectivity.Config{}

	flags.StringSliceVarP(
		&wco.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	flags.StringSliceVarP(
		&wco.InputFilePaths,
		"input-file",
		"f",
		[]string{},
		"path to file to supply test dependent input (may be specified multiple times)",
	)

	flags.StringSliceVarP(
		&wco.Inputs,
		"input",
		"i",
		[]string{},
		"add test-dependent input (may be specified multiple times)",
	)

	flags.Int64Var(
		&wco.MaxRuntime,
		"max-runtime",
		0,
		"maximum runtime in seconds for the experiment (zero means infinite)",
	)

	flags.BoolVar(
		&wco.Random,
		"random",
		false,
		"randomize the inputs list",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&wco.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
