package main

//
// Darwin build
//

import (
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// darwinSubcommand returns the darwin [cobra.Command].
func darwinSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "darwin",
		Short: "Builds ooniprobe and miniooni for darwin",
		Run: func(cmd *cobra.Command, args []string) {
			darwinBuildAll(&buildDeps{})
		},
		Args: cobra.NoArgs,
	}
}

// darwinBuildAll builds all the packages for darwin.
func darwinBuildAll(deps buildtoolmodel.Dependencies) {
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()
	archs := []string{"amd64", "arm64"}
	products := []*product{productMiniooni, productOoniprobe}
	for _, arch := range archs {
		for _, product := range products {
			darwinBuildPackage(deps, arch, product)
		}
	}
}

// darwinBuildPackagebuild builds a package for an architecture.
func darwinBuildPackage(deps buildtoolmodel.Dependencies, goarch string, product *product) {
	must.Fprintf(os.Stderr, "# building %s for darwin/%s\n", product.Pkg, goarch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append("-o", product.DestinationPath("darwin", goarch))
	argv.Append(product.Pkg)

	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "1")
	envp.Append("GOARCH", goarch)
	envp.Append("GOOS", "darwin")

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}
	runtimex.Try0(shellx.RunEx(config, argv, envp))

	must.Fprintf(os.Stderr, "\n")
}
