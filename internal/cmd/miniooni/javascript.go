package main

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/x/dsljavascript"
	"github.com/spf13/cobra"
)

// registerJavaScript registers the javascript subcommand
func registerJavaScript(rootCmd *cobra.Command, globalOptions *Options) {
	subCmd := &cobra.Command{
		Use:   "javascript",
		Short: "Very experimental command to run JavaScript snippets",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runtimex.Assert(len(args) == 1, "expected exactly one argument")
			javaScriptMain(args[0])
		},
	}
	rootCmd.AddCommand(subCmd)
}

func javaScriptMain(scriptPath string) {
	// TODO(bassosimone): for an initial prototype, using a local directory is
	// good, but, if we make this more production ready, we probably need to define
	// a specific location under the $OONI_HOME.
	config := &dsljavascript.VMConfig{
		Logger:        log.Log,
		ScriptBaseDir: filepath.Join(".", "./javascript"),
	}

	log.Warnf("The javascript subcommand is highly experimental and may be removed")
	log.Warnf("or heavily modified without prior notice. For more information, for now")
	log.Warnf("see https://github.com/bassosimone/2023-12-09-ooni-javascript.")

	runtimex.Try0(dsljavascript.RunScript(config, scriptPath))
}
