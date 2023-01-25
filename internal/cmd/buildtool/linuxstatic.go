package main

//
// Builds for Linux assuming static linking makes sense.
//

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// linuxStaticSubcommand returns the linuxStatic sucommand.
func linuxStaticSubcommand() *cobra.Command {
	config := &linuxStaticBuilder{
		goarm: 0,
	}
	cmd := &cobra.Command{
		Use:   "linux-static",
		Short: "Builds ooniprobe for linux assuming static linking is possible",
		Run:   config.main,
		Args:  cobra.NoArgs,
	}
	cmd.Flags().Int64Var(&config.goarm, "goarm", 0, "specifies the arm subarchitecture")
	return cmd
}

// linuxStaticBuilder is the build configuration.
type linuxStaticBuilder struct {
	goarm int64
}

// main is the main function of the linuxStatic subcommand.
func (b *linuxStaticBuilder) main(*cobra.Command, []string) {
	psiphonMaybeCopyConfigFiles()
	golangCheck()

	// TODO(bassosimone): I am running the container with the right userID but
	// apparently this is not enough to make git happy--why?
	must.Fprintf(os.Stderr, "# working around git file ownership checks\n")
	must.Run(log.Log, "git", "config", "--global", "--add", "safe.directory", "/ooni")
	must.Fprintf(os.Stderr, "\n")

	products := []*product{productMiniooni, productOoniprobe}
	for _, product := range products {
		b.build(product)
	}
}

// build runs the build.
func (b *linuxStaticBuilder) build(product *product) {
	cacheprefix := runtimex.Try1(filepath.Abs(filepath.Join("GOCACHE", "oonibuild", "v1", b.fullarch())))

	must.Fprintf(
		os.Stderr,
		"# building %s for linux/%s with static linking\n",
		product.Pkg,
		runtime.GOARCH,
	)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if psiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w -extldflags -static")
	argv.Append("-o", product.DestinationPath("linux", runtime.GOARCH))
	argv.Append(product.Pkg)

	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "1")
	envp.Append("GOOS", "linux")
	envp.Append("GOARCH", runtime.GOARCH)
	if b.goarm > 0 {
		envp.Append("GOARM", strconv.FormatInt(b.goarm, 10))
	}
	envp.Append("GOCACHE", filepath.Join(cacheprefix, "buildcache"))
	envp.Append("GOMODCACHE", filepath.Join(cacheprefix, "modcache"))

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	runtimex.Try0(shellx.RunEx(config, argv, envp))

	must.Fprintf(os.Stderr, "\n")
}

// fullarch returns the full arch name
func (cfg *linuxStaticBuilder) fullarch() string {
	switch runtime.GOARCH {
	case "arm":
		runtimex.Assert(cfg.goarm > 0, "expected a > 0 goarm value")
		return fmt.Sprintf("armv%d", cfg.goarm)
	default:
		return runtime.GOARCH
	}
}
