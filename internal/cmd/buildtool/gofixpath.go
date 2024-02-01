package main

import (
	"fmt"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// gofixpathSubcommand returns the gofixpath [cobra.Command].
func gofixpathSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gofixpath",
		Short: "Executes a command ensuring the expected version of Go comes first in PATH lookup",
		Run: func(cmd *cobra.Command, args []string) {
			gofixpathMain(&buildDeps{}, args...)
		},
		Args: cobra.MinimumNArgs(1),
	}
}

// gofixpathMain ensures the correct version of Go is in path, otherwise
// installs such a version, configure the PATH correctly, and then executes
// whatever argument passed to the command with the correct PATH.
//
// See https://github.com/ooni/probe/issues/2664.
func gofixpathMain(deps buildtoolmodel.Dependencies, args ...string) {
	// create empty environment
	envp := &shellx.Envp{}

	// install and configure the correct go version if needed
	if !golangCorrectVersionCheckP("GOVERSION") {
		// read the version of Go we would like to use
		expected := string(must.FirstLineBytes(must.ReadFile("GOVERSION")))

		// install the wrapper command
		packageName := fmt.Sprintf("golang.org/dl/go%s@latest", expected)
		must.Run(log.Log, "go", "install", "-v", packageName)

		// run the wrapper to download the distribution
		gobinproxy := filepath.Join(
			string(must.FirstLineBytes(must.RunOutput(log.Log, "go", "env", "GOPATH"))),
			"bin",
			fmt.Sprintf("go%s", expected),
		)
		must.Run(log.Log, gobinproxy, "download")

		// add the path to the SDK binary dir
		//
		// Note: because gomobile wants to execute "go" we must provide the
		// path to a directory that contains a command named "go" and we cannot
		// just use the gobinproxy binary
		sdkbinpath := filepath.Join(
			string(must.FirstLineBytes(must.RunOutput(log.Log, gobinproxy, "env", "GOROOT"))),
			"bin",
		)

		// prepend to PATH
		envp.Append("PATH", cdepsPrependToPath(sdkbinpath))
	}

	// create shellx configuration
	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	// create argv
	argv := runtimex.Try1(shellx.NewArgv(args[0], args[1:]...)) // safe because cobra.MinimumNArgs(1)

	// execute the child command
	runtimex.Try0(shellx.RunEx(config, argv, envp))
}
