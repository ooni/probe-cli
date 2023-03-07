package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type signalOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &signalOptions{}

func init() {
	options := &signalOptions{}
	AllExperimentOptions["signal"] = options
	AllExperiments["signal"] = &signal.ExperimentMain{}
}

// SetArguments
func (so *signalOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(so.Annotations)
	return &model.ExperimentMainArgs{
		Annotations:    annotations,
		CategoryCodes:  nil, // accept any category
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
func (so *signalOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(so.ConfigOptions)
}

// BuildWithOONIRun
func (so *signalOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(so, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (so *signalOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &signal.Config{}

	flags.StringSliceVarP(
		&so.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&so.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
