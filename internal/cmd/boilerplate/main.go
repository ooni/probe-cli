// Command boilerplate assists you in generating code for new experiments.
//
// We will generate experiments under the ./internal/experiment folder rather
// than under ./internal/engine/experiment because we are moving away from the
// experiment folder.
package main

import (
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "boilerplate",
		Short: "Helps to auto-generate code for new experiments",
	}

	newExperiment := &cobra.Command{
		Use:   "new-experiment",
		Args:  cobra.NoArgs,
		Short: "Interactively generate a new experiment",
		Run:   (&NewExperimentCommand{}).Run,
	}
	root.AddCommand(newExperiment)

	newflow := &cobra.Command{
		Use:   "new-task",
		Args:  cobra.NoArgs,
		Short: "Interactively generate a new task for an experiment",
		Run:   (&NewTaskCommand{}).Run,
	}
	root.AddCommand(newflow)

	err := root.Execute()
	runtimex.PanicOnError(err, "root.Execute failed")
}
