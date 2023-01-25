package main

//
// Generic builder for the current GOOS/GOARCH
//

import (
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// genericSubcommand returns the generic sucommand.
func genericSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Generic Go builder for the current GOOS and GOARCH",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "miniooni",
		Short: "Builds miniooni for the current GOOS and GOARCH",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericBuildPackage(&buildDeps{}, productMiniooni)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "ooniprobe",
		Short: "Builds ooniprobe for the current GOOS and GOARCH",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericBuildPackage(&buildDeps{}, productOoniprobe)
		},
	})
	return cmd
}

// genericBuildPackage is the generic function for building a package.
func genericBuildPackage(deps buildtoolmodel.Dependencies, product *product) {
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	log.Infof("building %s for %s/%s", product.Pkg, runtime.GOOS, runtime.GOARCH)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append(product.Pkg)

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	runtimex.Try0(shellx.RunEx(config, argv, &shellx.Envp{}))
}
