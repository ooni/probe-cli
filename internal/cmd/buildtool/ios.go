package main

//
// iOS builds
//

import (
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// iosSubcommand returns the ios [cobra.Command].
func iosSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ios",
		Short: "Builds oonimkall for iOS",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "gomobile",
		Short: "Builds oonimkall for iOS using gomobile",
		Run: func(cmd *cobra.Command, args []string) {
			iosBuildGomobile(&buildDeps{})
		},
	})
	return cmd
}

// iosBuildGomobile invokes the gomobile build.
func iosBuildGomobile(deps buildtoolmodel.Dependencies) {
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	config := &gomobileConfig{
		deps:       deps,
		envp:       &shellx.Envp{},
		extraFlags: []string{},
		output:     filepath.Join("MOBILE", "ios", "oonimkall.xcframework"),
		target:     "ios",
	}
	log.Info("building the mobile library using gomobile")
	gomobileBuild(config)
}
