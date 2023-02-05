package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/hirl"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type hirlOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &hirlOptions{}

func init() {
	options := &hirlOptions{}
	AllExperimentOptions["hirl"] = options
	AllExperiments["hirl"] = &hirl.ExperimentMain{}
}

// SetArguments
func (hirlo *hirlOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(hirlo.Annotations)
	return &model.ExperimentMainArgs{
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
}

// ExtraOptions
func (hirlo *hirlOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(hirlo.ConfigOptions)
}

// BuildWithOONIRun
func (hirlo *hirlOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(hirlo, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (hirlo *hirlOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := hirl.Config{}

	flags.StringSliceVarP(
		&hirlo.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&hirlo.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
