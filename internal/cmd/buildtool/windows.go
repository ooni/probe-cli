package main

//
// Windows build
//

import (
	"errors"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// windowsSubcommand returns the windows sucommand.
func windowsSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "windows",
		Short: "Builds ooniprobe for windows",
		Run: func(cmd *cobra.Command, args []string) {
			windowsBuildAll(&buildDeps{})
		},
		Args: cobra.NoArgs,
	}
}

// windowsBuildAll is the main function of the windows subcommand.
func windowsBuildAll(deps buildtoolmodel.Dependencies) {
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()
	deps.WindowsMingwCheck()
	archs := []string{"386", "amd64"}
	products := []*product{productMiniooni, productOoniprobe}
	for _, arch := range archs {
		for _, product := range products {
			windowsBuildPackage(deps, arch, product)
		}
	}
}

// windowsBuildPackage builds the given package for windows
// compiling for the specified architecture.
func windowsBuildPackage(deps buildtoolmodel.Dependencies, goarch string, product *product) {
	log.Infof("building %s for windows/%s", product.Pkg, goarch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}

	argv.Append("-ldflags", "-s -w")
	argv.Append("-o", product.DestinationPath("windows", goarch))
	argv.Append(product.Pkg)

	envp := &shellx.Envp{}
	switch goarch {
	case "amd64":
		envp.Append("CC", windowsMingwAmd64Compiler)
	case "386":
		envp.Append("CC", windowsMingw386Compiler)
	default:
		panic(errors.New("unsupported windows goarch"))
	}

	envp.Append("CGO_ENABLED", "1")
	envp.Append("GOARCH", goarch)
	envp.Append("GOOS", "windows")

	config := &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}

	runtimex.Try0(shellx.RunEx(config, argv, envp))
}

// windowsMingwExpectedVersion is the expected version of mingw-w64,
// which may be overriden by setting the EXPECTED_MINGW_W64_VERSION
// environment variable before starting the build.
var windowsMingwExpectedVersion = "12.2.0"

// windowsMingwEnvironmentVariable is the name of the environment variable
// that overrides the expected mingw version.
const windowsMingwEnvironmentVariable = "EXPECTED_MINGW_W64_VERSION"

// windowsMingwAmd64Compiler is the amd64 compiler.
const windowsMingwAmd64Compiler = "x86_64-w64-mingw32-gcc"

// windowsMingw386Compiler is the 386 compiler.
const windowsMingw386Compiler = "i686-w64-mingw32-gcc"

// windowsMingwCheck checks we're using the correct mingw version.
func windowsMingwCheck() {
	windowsMingwCheckFor(windowsMingwAmd64Compiler)
	windowsMingwCheckFor(windowsMingw386Compiler)
}

// windowsMingwCheckFor implements mingwCheck for the given compiler.
func windowsMingwCheckFor(compiler string) {
	expected := windowsMingwExpectedVersionGetter()
	firstLine := string(must.FirstLineBytes(must.RunOutputQuiet(compiler, "--version")))
	v := strings.Split(firstLine, " ")
	runtimex.Assert(len(v) == 3, "expected to see exactly three tokens")
	if got := v[2]; got != expected {
		log.Fatalf("expected mingw %s but got %s", expected, got)
	}
	log.Infof("using %s %s", compiler, expected)
}

// windowsMingwEexpectedVersionGetter returns the correct expected mingw version.
func windowsMingwExpectedVersionGetter() string {
	value := os.Getenv(windowsMingwEnvironmentVariable)
	if value == "" {
		return windowsMingwExpectedVersion
	}
	log.Infof("mingw version overriden using %s", windowsMingwEnvironmentVariable)
	return value
}
