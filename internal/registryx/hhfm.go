package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/hhfm"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

// hhfmOptions contains options for hhfm.
type hhfmOptions struct {
	Annotations    []string
	InputFilePaths []string
	Inputs         []string
	MaxRuntime     int64
	Random         bool
	ConfigOptions  []string
}

var _ model.ExperimentOptions = &hhfmOptions{}

func init() {
	options := &hhfmOptions{}
	AllExperimentOptions["hhfm"] = options
	AllExperiments["hhfm"] = &hhfm.ExperimentMain{}
}

func (hhfmo *hhfmOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(hhfmo.Annotations)
	return &model.ExperimentMainArgs{
		Annotations:    annotations, // TODO(bassosimone): fill
		CategoryCodes:  nil,         // accept any category
		Charging:       true,
		Callbacks:      model.NewPrinterCallbacks(log.Log),
		Database:       db.Database,
		Inputs:         hhfmo.Inputs,
		MaxRuntime:     hhfmo.MaxRuntime,
		MeasurementDir: db.DatabaseResult.MeasurementDir,
		NoCollector:    false,
		OnWiFi:         true,
		ResultID:       db.DatabaseResult.ID,
		RunType:        model.RunTypeManual,
		Session:        sess,
	}
}

// ExtraOptions
func (hhfmo *hhfmOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(hhfmo.ConfigOptions)
}

// BuildWithOONIRun
func (hhfmo *hhfmOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(hhfmo, args); err != nil {
		return err
	}
	return nil
}

func (hhfmo *hhfmOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := hhfm.Config{}

	flags.StringSliceVarP(
		&hhfmo.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	flags.StringSliceVarP(
		&hhfmo.InputFilePaths,
		"input-file",
		"f",
		[]string{},
		"path to file to supply test dependent input (may be specified multiple times)",
	)

	flags.StringSliceVarP(
		&hhfmo.Inputs,
		"input",
		"i",
		[]string{},
		"add test-dependent input (may be specified multiple times)",
	)

	flags.Int64Var(
		&hhfmo.MaxRuntime,
		"max-runtime",
		0,
		"maximum runtime in seconds for the experiment (zero means infinite)",
	)

	flags.BoolVar(
		&hhfmo.Random,
		"random",
		false,
		"randomize the inputs list",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&hhfmo.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
