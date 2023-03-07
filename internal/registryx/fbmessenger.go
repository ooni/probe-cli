package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type fbmessengerOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &fbmessengerOptions{}

func init() {
	options := &fbmessengerOptions{}
	AllExperimentOptions["facebook_messenger"] = options
	AllExperiments["facebook_messenger"] = &fbmessenger.ExperimentMain{}
}

// SetAruments
func (fbo *fbmessengerOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(fbo.Annotations)
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
func (fbo *fbmessengerOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(fbo.ConfigOptions)
}

// BuildWithOONIRun
func (fbo *fbmessengerOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(fbo, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (fbo *fbmessengerOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := fbmessenger.Config{}

	flags.StringSliceVarP(
		&fbo.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&fbo.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
