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
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
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
	linuxStaticBuilAll(&buildDependencies{}, runtime.GOARCH, b.goarm)
}

// linuxStaticBuildAll builds all the packages on a linux-static environment.
func linuxStaticBuilAll(deps buildtoolmodel.Dependencies, goarch string, goarm int64) {
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	// TODO(bassosimone): I am running the container with the right userID but
	// apparently this is not enough to make git happy--why?
	must.Fprintf(os.Stderr, "# working around git file ownership checks\n")
	must.Run(log.Log, "git", "config", "--global", "--add", "safe.directory", "/ooni")
	must.Fprintf(os.Stderr, "\n")

	products := []*product{productMiniooni, productOoniprobe}
	cacheprefix := runtimex.Try1(filepath.Abs("GOCACHE"))
	for _, product := range products {
		linuxStaticBuildPackage(deps, product, goarch, goarm, cacheprefix)
	}
}

// linuxStaticBuildPackage builds a package in a linux static environment.
func linuxStaticBuildPackage(
	deps buildtoolmodel.Dependencies,
	product *product,
	goarch string,
	goarm int64,
	cacheprefix string,
) {
	must.Fprintf(
		os.Stderr,
		"# building %s for linux/%s with static linking\n",
		product.Pkg,
		goarch,
	)

	ooniArch := linuxStaticBuildOONIArch(goarch, goarm)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w -extldflags -static")
	argv.Append("-o", product.DestinationPath("linux", ooniArch))
	argv.Append(product.Pkg)

	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "1")
	envp.Append("GOOS", "linux")
	envp.Append("GOARCH", goarch)
	if goarm > 0 {
		envp.Append("GOARM", strconv.FormatInt(goarm, 10))
	}
	cachedirbase := filepath.Join(
		cacheprefix,
		"oonibuild",
		"v1",
		ooniArch,
	)
	envp.Append("GOCACHE", filepath.Join(cachedirbase, "buildcache"))
	envp.Append("GOMODCACHE", filepath.Join(cachedirbase, "modcache"))

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	runtimex.Try0(shellx.RunEx(config, argv, envp))

	must.Fprintf(os.Stderr, "\n")
}

// linuxStaticBuildOONIArch returns the OONI arch name. This is equal
// to the GOARCH but for arm where we use armv6 and armv7.
func linuxStaticBuildOONIArch(goarch string, goarm int64) string {
	switch goarch {
	case "arm":
		runtimex.Assert(goarm > 0, "expected a > 0 goarm value")
		return fmt.Sprintf("armv%d", goarm)
	default:
		return goarch
	}
}
