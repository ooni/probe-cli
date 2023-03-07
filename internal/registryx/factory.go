package registryx

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

// Experiment
type Experiment interface {
	//
	Main(ctx context.Context) error

	//
	SetOptions(options model.ExperimentOptions)

	//
	SetArguments(sess model.ExperimentSession, db *model.DatabaseProps, extraOptions map[string]any) error
}

// Factory is a forwarder for the respective experiment's main
type Factory struct {
	// BuildFlags initializes the experiment specific flags
	BuildFlags func(experimentName string, rootCmd *cobra.Command) model.ExperimentOptions
}
