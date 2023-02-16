package main

//
// Builds oohelperd for linux/amd64
//

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// oohelperdSubcommand returns the oohelperd sucommand.
func oohelperdSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "oohelperd",
		Short: "Build and deploy oohelperd",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "build",
		Short: "Builds oohelperd for linux/amd64",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			oohelperdBuildAndMaybeDeploy(&buildDeps{}, false)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "deploy",
		Short: "Builds and deploys oohelperd on 0.th.ooni.org",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			oohelperdBuildAndMaybeDeploy(&buildDeps{}, true)
		},
	})
	return cmd
}

// oohelperdBuildAndMaybeDeploy builds oohelperd for linux/amd64 and
// possibly deploys the build to the 0.th.ooni.org server.
func oohelperdBuildAndMaybeDeploy(deps buildtoolmodel.Dependencies, deploy bool) {
	deps.GolangCheck()

	log.Info("building oohelperd for linux/amd64")
	oohelperdBinary := filepath.Join("CLI", "oohelperd-linux-amd64")

	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "0")
	envp.Append("GOOS", "linux")
	envp.Append("GOARCH", "amd64")

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	argv.Append("-o", oohelperdBinary)
	argv.Append("-tags", "netgo")
	argv.Append("-ldflags", "-s -w -extldflags -static")
	argv.Append("./internal/cmd/oohelperd")

	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))

	if deploy {
		runtimex.Try0(shellx.Run(log.Log, "scp", oohelperdBinary, "0.th.ooni.org:oohelperd"))
	}
}
