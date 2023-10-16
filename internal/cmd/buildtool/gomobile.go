package main

//
// Common code for gomobile based builds.
//

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// gomobileConfig contains the config for gomobileBuild.
type gomobileConfig struct {
	// deps contains the build dependencies.
	deps buildtoolmodel.Dependencies

	// envp is the environment to use.
	envp *shellx.Envp

	// extraFlags contains extra flags for the gomobile bind command.
	extraFlags []string

	// output is the name of the output file.
	output string

	// target is the build target (e.g. "android").
	target string
}

// gomobileBuild runs a build based on gomobile.
func gomobileBuild(config *gomobileConfig) {
	// Undoes the effects of go-getting golang.org/x/mobile/cmd/gomobile
	defer must.Run(log.Log, "go", "mod", "tidy")

	must.Run(log.Log, "go", "install", "golang.org/x/mobile/cmd/gomobile@latest")

	gopath := config.deps.GOPATH()
	gomobile := filepath.Join(gopath, "bin", "gomobile")
	must.Run(log.Log, gomobile, "init")

	// Adding gomobile to go.mod as documented by golang.org/wiki/Mobile
	must.Run(log.Log, "go", "get", "-d", "golang.org/x/mobile/cmd/gomobile")

	argv := runtimex.Try1(shellx.NewArgv(gomobile, "bind"))
	argv.Append("-target", config.target)
	argv.Append("-o", config.output)
	for _, entry := range config.extraFlags {
		argv.Append(entry)
	}
	if config.deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config,ooni_libtor")
	} else {
		argv.Append("-tags", "ooni_libtor")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append("./pkg/oonimkall")

	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, config.envp))
}
