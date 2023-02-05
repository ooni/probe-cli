package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type riseupvpnOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &riseupvpnOptions{}

func init() {
	options := &riseupvpnOptions{}
	AllExperimentOptions["riseupvpn"] = options
	AllExperiments["riseupvpn"] = &riseupvpn.ExperimentMain{}
}

// SetArguments
func (ro *riseupvpnOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(ro.Annotations)
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
func (ro *riseupvpnOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(ro.ConfigOptions)
}

// BuildWithOONIRun
func (ro *riseupvpnOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(ro, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (ro *riseupvpnOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &riseupvpn.Config{}

	flags.StringSliceVarP(
		&ro.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&ro.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
