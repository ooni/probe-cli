package main

//
// Main
//

import (
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

func main() {
	go func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %+v\n", r)
			os.Exit(1)
		}
	}()
	root := &cobra.Command{
		Use:   "buildtool",
		Short: "Tool for building ooniprobe",
	}
	root.AddCommand(darwinSubcommand())
	err := root.Execute()
	runtimex.PanicOnError(err, "root.Execute")
}
