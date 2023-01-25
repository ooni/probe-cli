package main

//
// Generic builder for the current GOOS/GOARCH
//

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// genericSubcommand returns the generic sucommand.
func genericSubcommand(p *product) *cobra.Command {
	name := filepath.Base(p.Pkg)
	cfg := &genericBuilder{
		p: p,
	}
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Builds %s for %s/%s", name, runtime.GOOS, runtime.GOARCH),
		Run:   cfg.main,
		Args:  cobra.NoArgs,
	}
}

// genericBuilder is the configuration for a generic build
type genericBuilder struct {
	p *product
}

// main is the main function of the generic subcommand.
func (b *genericBuilder) main(*cobra.Command, []string) {
	psiphonMaybeCopyConfigFiles()
	golangCheck()
	hasPsiphon := psiphonFilesExist()
	genericBuildPackage(b.p, hasPsiphon)
}

// genericBuildPackage is the generic function for building a package.
func genericBuildPackage(product *product, hasPsiphon bool) {
	must.Fprintf(
		os.Stderr,
		"# building %s for %s/%s\n",
		product.Pkg,
		runtime.GOOS,
		runtime.GOARCH,
	)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if hasPsiphon {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append(product.Pkg)

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	runtimex.Try0(shellx.RunEx(config, argv, &shellx.Envp{}))

	must.Fprintf(os.Stderr, "\n")
}
