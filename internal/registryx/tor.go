package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/tor"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type torOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &torOptions{}

func init() {
	options := &webConnectivityOptions{}
	AllExperimentOptions["tor"] = options
	AllExperiments["tor"] = &tor.ExperimentMain{}
}

func (toro *torOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(toro.Annotations)
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

func (toro *torOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(toro.ConfigOptions)
}

func (toro *torOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(toro, args); err != nil {
		return err
	}
	return nil
}

func (toro *torOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &tor.Config{}

	flags.StringSliceVarP(
		&toro.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&toro.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
