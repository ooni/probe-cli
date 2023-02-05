package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type dashOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &dashOptions{}

func init() {
	options := &dashOptions{}
	AllExperimentOptions["dash"] = options
	AllExperiments["dash"] = &dash.ExperimentMain{}
}

// SetArguments
func (do *dashOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(do.Annotations)
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
func (do *dashOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(do.ConfigOptions)
}

// BuildWithOONIRun
func (do *dashOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(do, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (do *dashOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := dash.Config{}

	flags.StringSliceVarP(
		&do.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&do.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
