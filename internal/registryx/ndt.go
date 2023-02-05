package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/ndt7"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type ndtOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &ndtOptions{}

func init() {
	options := &ndtOptions{}
	AllExperimentOptions["ndt"] = options
	AllExperiments["ndt"] = &ndt7.ExperimentMain{}
}

// SetArguments
func (ndto *ndtOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(ndto.Annotations)
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
func (ndto *ndtOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(ndto.ConfigOptions)
}

// BuildWithOONIRun
func (ndto *ndtOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(ndto, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (ndto *ndtOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := ndt7.Config{}

	flags.StringSliceVarP(
		&ndto.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&ndto.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
