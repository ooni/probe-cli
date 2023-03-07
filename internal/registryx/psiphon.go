package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/psiphon"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type psiphonOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &psiphonOptions{}

func init() {
	options := &psiphonOptions{}
	AllExperimentOptions["psiphon"] = options
	AllExperiments["psiphon"] = &psiphon.ExperimentMain{}
}

// SetArguments
func (po *psiphonOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(po.Annotations)
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
func (po *psiphonOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(po.ConfigOptions)
}

// BuildWithOONIRun
func (po *psiphonOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(po, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (po *psiphonOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := psiphon.Config{}

	flags.StringSliceVarP(
		&po.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&po.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
