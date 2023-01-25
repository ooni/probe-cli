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
	builder := &darwinBuilder{}
	return &cobra.Command{
		Use:   "darwin",
		Short: "Builds ooniprobe for darwin",
		Run:   builder.main,
		Args:  cobra.NoArgs,
	}
}

// darwinBuilder builds for darwin.
type darwinBuilder struct{}

// main is the main function of the darwin subcommand.
func (b *darwinBuilder) main(cmd *cobra.Command, args []string) {
	psiphonMaybeCopyConfigFiles()
	golangCheck()
	archs := []string{"amd64", "arm64"}
	products := []*product{productMiniooni, productOoniprobe}
	for _, arch := range archs {
		for _, product := range products {
			b.build(arch, product)
		}
	}
}

// build builds the given package for darwin compiling for the specified architecture.
func (b *darwinBuilder) build(goarch string, product *product) {
	must.Fprintf(os.Stderr, "# building %s for darwin/%s\n", product.Pkg, goarch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if psiphonFilesExist() {
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
