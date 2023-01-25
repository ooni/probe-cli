package main

//
// Darwin build
//

import (
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// darwinSubcommand returns the darwin sucommand.
func darwinSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "darwin",
		Short: "Builds ooniprobe for darwin",
		Run: func(cmd *cobra.Command, args []string) {
			darwinBuildAll(&buildDependencies{})
		},
		Args: cobra.NoArgs,
	}
}

// darwinBuildAll builds all the possible packages for darwin.
func darwinBuildAll(deps buildDeps) {
	deps.psiphonMaybeCopyConfigFiles()
	deps.golangCheck()
	archs := []string{"amd64", "arm64"}
	products := []*product{productMiniooni, productOoniprobe}
	for _, arch := range archs {
		for _, product := range products {
			darwinBuildPackage(deps, arch, product)
		}
	}
}

// darwinBuildPackagebuild builds the given package for darwin
// compiling for the specified architecture.
func darwinBuildPackage(deps buildDeps, goarch string, product *product) {
	must.Fprintf(os.Stderr, "# building %s for darwin/%s\n", product.Pkg, goarch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.psiphonFilesExist() {
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
