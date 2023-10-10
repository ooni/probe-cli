package main

//
// iOS builds
//

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// iosSubcommand returns the ios [cobra.Command].
func iosSubcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ios",
		Short: "Builds oonimkall and its dependencies for iOS",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "gomobile",
		Short: "Builds oonimkall for iOS using gomobile",
		Run: func(cmd *cobra.Command, args []string) {
			iosBuildGomobile(&buildDeps{})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "cdeps {zlib|openssl|libevent|tor} [zlib|openssl|libevent|tor...]",
		Short: "Cross compiles C dependencies for iOS",
		Run: func(cmd *cobra.Command, args []string) {
			for _, arg := range args {
				iosCdepsBuildMain(arg, &buildDeps{})
			}
		},
		Args: cobra.MinimumNArgs(1),
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

// iosCdepsBuildMain builds C dependencies for ios.
func iosCdepsBuildMain(name string, deps buildtoolmodel.Dependencies) {
	runtimex.Assert(
		runtime.GOOS == "darwin" || runtime.GOOS == "linux",
		"this command requires darwin or linux",
	)
	archs := []string{"arm", "arm64", "386", "amd64"}
	for _, arch := range archs {
		iosCdepsBuildArch(deps, arch, name)
	}
}

// iosNewCBuildEnv creates a new [cBuildEnv] for the given ooniArch ("arm", "arm64", "386", "amd64").
func iosNewCBuildEnv(ooniArch string) *cBuildEnv {
	destdir := runtimex.Try1(filepath.Abs(filepath.Join( // must be absolute
		"internal", "libtor", "ios", ooniArch,
	)))
	//     export CFLAGS="-O3 -arch armv7 -arch armv7s -arch arm64 -isysroot $XCODE_ROOT/Platforms/iPhoneOS.platform/Developer/SDKs/iPhoneOS${IPHONE_SDKVERSION}.sdk -mios-version-min=${IPHONE_SDKVERSION} -fembed-bitcode"
	out := &cBuildEnv{
		ANDROID_HOME:       "",
		ANDROID_NDK_ROOT:   "",
		AS:                 "", // later
		AR:                 "",
		BINPATH:            "",
		CC:                 "clang",
		CFLAGS:             []string{},
		CONFIGURE_HOST:     "", // later
		DESTDIR:            destdir,
		CXX:                "clang++",
		CXXFLAGS:           []string{},
		GOARCH:             ooniArch,
		GOARM:              "", // maybe later
		LD:                 "",
		LDFLAGS:            []string{}, // empty
		OPENSSL_API_DEFINE: "",
		OPENSSL_COMPILER:   "", // later
		RANLIB:             "",
		STRIP:              "",
	}
	switch ooniArch {
	case "arm":
		out.CFLAGS = []string{
			"-arch", "armv7",
			"-isysroot", "/Library/Developer/CommandLineTools/"
		}
		/*
			out.CC = filepath.Join(out.BINPATH, "armv7a-linux-androideabi21-clang")
			out.CXX = filepath.Join(out.BINPATH, "armv7a-linux-androideabi21-clang++")
			out.GOARM = "7"
			out.CONFIGURE_HOST = "arm-linux-androideabi"
			out.OPENSSL_COMPILER = "android-arm"
		*/
	case "arm64":
		/*
			out.CC = filepath.Join(out.BINPATH, "aarch64-linux-android21-clang")
			out.CXX = filepath.Join(out.BINPATH, "aarch64-linux-android21-clang++")
			out.CONFIGURE_HOST = "aarch64-linux-android"
			out.OPENSSL_COMPILER = "android-arm64"
		*/
	case "386":
		/*
			out.CC = filepath.Join(out.BINPATH, "i686-linux-android21-clang")
			out.CXX = filepath.Join(out.BINPATH, "i686-linux-android21-clang++")
			out.CONFIGURE_HOST = "i686-linux-android"
			out.OPENSSL_COMPILER = "android-x86"
		*/
	case "amd64":
		/*
			out.CC = filepath.Join(out.BINPATH, "x86_64-linux-android21-clang")
			out.CXX = filepath.Join(out.BINPATH, "x86_64-linux-android21-clang++")
			out.CONFIGURE_HOST = "x86_64-linux-android"
			out.OPENSSL_COMPILER = "android-x86_64"
		*/
	default:
		panic(errors.New("unsupported ooniArch"))
	}
	out.AS = out.CC
	return out
}

// iosCdepsBuildArch builds the given dependency for the given arch
func iosCdepsBuildArch(deps buildtoolmodel.Dependencies, arch string, name string) {
	cdenv := iosNewCBuildEnv(arch)
	switch name {
	case "libevent":
		cdepsLibeventBuildMain(cdenv, deps)
	case "openssl":
		cdepsOpenSSLBuildMain(cdenv, deps)
	case "tor":
		cdepsTorBuildMain(cdenv, deps)
	case "zlib":
		cdepsZlibBuildMain(cdenv, deps)
	default:
		panic(fmt.Errorf("unknown dependency: %s", name))
	}
}
