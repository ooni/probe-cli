package main

//
// Android build
//

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// androidSubcommand returns the android [cobra.Command].
func androidSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "android",
		Short: "Builds ooniprobe and miniooni for android",
		Run: func(cmd *cobra.Command, args []string) {
			androidBuildAll(&buildDeps{})
		},
	}
	return cmd
}

// androidBuildAll is the main function of the android subcommand.
func androidBuildAll(deps buildtoolmodel.Dependencies) {
	runtimex.Assert(
		runtime.GOOS == "linux" || runtime.GOOS == "android",
		"this command requires darwin or linux",
	)

	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()
	deps.AndroidSDKCheck()

	androidHome := deps.AndroidSDKCheck()
	ndkDir := deps.AndroidNDKCheck(androidHome)

	envp := &shellx.Envp{}
	envp.Append("ANDROID_HOME", androidHome)
	envp.Append("ANDROID_NDK_HOME", ndkDir)

	androidBuildGomobile(deps, envp)
}

// androidBuildGomobile invokes the gomobile build.
func androidBuildGomobile(deps buildtoolmodel.Dependencies, envp *shellx.Envp) {
	config := &gomobileConfig{
		deps:       deps,
		envp:       envp,
		extraFlags: []string{"-androidapi", "21"},
		output:     filepath.Join("MOBILE", "android", "oonimkall.aar"),
		target:     "android",
	}
	log.Info("building the mobile library using gomobile")
	gomobileBuild(config)
}

// androidSDKCheck checks we have the right SDK installed.
func androidSDKCheck() string {
	// Make sure we have a working ANDROID_HOME
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		switch runtime.GOOS {
		case "darwin":
			androidHome = os.ExpandEnv("${HOME}/Library/Android/sdk")
		case "linux":
			androidHome = os.ExpandEnv("${HOME}/Android/Sdk")
		default:
			panic(errors.New("unsupported runtime.GOOS"))
		}
	}
	if !fsx.DirectoryExists(androidHome) {
		log.Warnf("expected to find Android SDK at %s, but found nothing", androidHome)
		log.Infof("HINT: run ./MOBILE/android/setup to (re)install the SDK")
		log.Fatalf("cannot continue without a valid Android SDK installation")
	}
	return androidHome
}

// androidNDKCheck checks we have the right NDK version.
func androidNDKCheck(androidHome string) string {
	ndkVersion := string(must.FirstLineBytes(must.ReadFile("NDKVERSION")))
	ndkDir := filepath.Join(androidHome, "ndk", ndkVersion)
	if !fsx.DirectoryExists(ndkDir) {
		log.Warnf("expected to find Android NDK at %s, but found nothing", ndkDir)
		log.Infof("HINT: run ./MOBILE/android/setup to (re)install the SDK")
		log.Fatalf("cannot continue without a valid Android NDK installation")
	}
	return ndkDir
}
