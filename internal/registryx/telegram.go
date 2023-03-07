package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type telegramOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &telegramOptions{}

func init() {
	options := &telegramOptions{}
	AllExperimentOptions["telegram"] = options
	AllExperiments["telegram"] = &telegram.ExperimentMain{}
}

// SetArguments
func (to *telegramOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(to.Annotations)
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
func (to *telegramOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(to.ConfigOptions)
}

// BuildWithOONIRun
func (to *telegramOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(to, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (to *telegramOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &telegram.Config{}

	flags.StringSliceVarP(
		&to.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&to.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
