package main

import (
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// desktopSubcommand returns the desktop [cobra.Command].
func desktopSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Builds oonimkall and its dependencies for desktop",
	}

	var targetOs string

	oomobileCmd := &cobra.Command{
		Use:   "oomobile",
		Short: "Builds oonimkall for desktop using oomobile",
		Run: func(cmd *cobra.Command, args []string) {
			desktopBuildOomobile(&buildDeps{}, targetOs)
		},
	}

	oomobileCmd.Flags().StringVar(&targetOs, "target", "linux", "Target OS (e.g., linux, windows, darwin)")

	cmd.AddCommand(oomobileCmd)
	return cmd
}

// desktopBuildOomobile invokes the oomobile build.
func desktopBuildOomobile(deps buildtoolmodel.Dependencies, targetOs string) {
	deps.GolangCheck()

	config := &gomobileConfig{
		deps:       deps,
		envp:       &shellx.Envp{},
		extraFlags: []string{},
		output:     filepath.Join("DESKTOP", "oonimkall.jar"),
		target:     "java",
	}
	config.envp.Append("GOOS", targetOs)

	// NOTE: we only support windows builds on amd64 for now
	if targetOs == "windows" {
		log.Infof("detected GOOS: %s, setting target as amd64", runtime.GOOS)
		config.target = "java/amd64"
		config.envp.Append("CC", "x86_64-w64-mingw32-gcc")
	}

	log.Info("building the desktop jar using oomobile")
	oomobileBuild(config)
}
