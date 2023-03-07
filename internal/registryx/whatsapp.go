package registryx

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/setter"
	"github.com/spf13/cobra"
)

type whatsappOptions struct {
	Annotations   []string
	ConfigOptions []string
}

var _ model.ExperimentOptions = &whatsappOptions{}

func init() {
	options := &whatsappOptions{}
	AllExperimentOptions["whatsapp"] = options
	AllExperiments["whatsapp"] = &whatsapp.ExperimentMain{}
}

// SetArguments
func (wo *whatsappOptions) SetArguments(sess model.ExperimentSession,
	db *model.DatabaseProps) *model.ExperimentMainArgs {
	annotations := mustMakeMapStringString(wo.Annotations)
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
func (wo *whatsappOptions) ExtraOptions() map[string]any {
	return mustMakeMapStringAny(wo.ConfigOptions)
}

// BuildWithOONIRun
func (wo *whatsappOptions) BuildWithOONIRun(inputs []string, args map[string]any) error {
	if err := setter.SetOptionsAny(wo, args); err != nil {
		return err
	}
	return nil
}

// BuildFlags
func (wo *whatsappOptions) BuildFlags(experimentName string, rootCmd *cobra.Command) {
	flags := rootCmd.Flags()
	config := &whatsapp.Config{}

	flags.StringSliceVarP(
		&wo.Annotations,
		"annotation",
		"A",
		[]string{},
		"add KEY=VALUE annotation to the report (can be repeated multiple times)",
	)

	if doc := setter.DocumentationForOptions(experimentName, config); doc != "" {
		flags.StringSliceVarP(
			&wo.ConfigOptions,
			"options",
			"O",
			[]string{},
			doc,
		)
	}
}
