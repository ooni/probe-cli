package main

//
// Generic builder for the current GOOS/GOARCH
//

import (
	"fmt"
	"path"
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
	cmd.AddCommand(&cobra.Command{
		Use:   "libooniengine",
		Short: "Builds libooniengine for the current GOOS and GOARCH",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			genericBuildLibrary(&buildDeps{}, productLibooniengine)
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

	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, &shellx.Envp{}))
}

// genericBuildLibrary is the generic function for building a library.
func genericBuildLibrary(deps buildtoolmodel.Dependencies, product *product) {
	deps.GolangCheck()
	os := deps.GOOS()

	log.Infof("building %s for %s/%s", product.Pkg, os, runtime.GOARCH)

	lib := path.Base(product.Pkg)
	library, err := generateLibrary(lib, os)
	runtimex.PanicOnError(err, fmt.Sprintf("failed to build for %s", os))

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	argv.Append("-buildmode", "c-shared")
	argv.Append("-o", library)
	argv.Append(product.Pkg)

	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, &shellx.Envp{}))
}
