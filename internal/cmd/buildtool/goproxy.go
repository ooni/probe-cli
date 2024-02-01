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

// goproxySubcommand returns the goproxy [cobra.Command].
func goproxySubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "goproxy",
		Short: "Executes a command ensuring the correct version of Go is in path",
		Run: func(cmd *cobra.Command, args []string) {
			goproxyMain(&buildDeps{}, args...)
		},
		Args: cobra.MinimumNArgs(1),
	}
}

// goproxyMain ensures the correct version of Go is in path, otherwise
// installs this version, configure the PATH correctly, and then executes
// whatever argument passed to the command with the correct path.
func goproxyMain(deps buildtoolmodel.Dependencies, args ...string) {
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
