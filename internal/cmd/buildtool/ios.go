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
	"github.com/ooni/probe-cli/v3/internal/must"
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
		runtime.GOOS == "darwin",
		"this command requires darwin or linux",
	)

	// The assembly of the arm version is broken for unknown reasons
	//archs := []string{"arm", "arm64", "386", "amd64"}
	// It seems there's no support for 386?
	//archs := []string{"arm64", "386", "amd64"}
	archs := []string{"arm64", "amd64"}
	for _, arch := range archs {
		iosCdepsBuildArch(deps, arch, name)
	}
}

// iosPlatformForOONIArch maps the ooniArch to the iOS platform
var iosPlatformForOONIArch = map[string]string{
	"386":   "iphonesimulator",
	"amd64": "iphonesimulator",
	"arm":   "iphoneos",
	"arm64": "iphoneos",
}

// iosAppleArchForOONIArch maps the ooniArch to the corresponding apple arch
var iosAppleArchForOONIArch = map[string]string{
	"386":   "i386",
	"amd64": "x86_64",
	"arm":   "armv7s",
	"arm64": "arm64",
}

// iosMinVersionFlagForOONIArch maps the ooniArch to the corresponding compiler flag
// to set the minimum version of either iphoneos or iphonesimulator.
//
// TODO(bassosimone): the OpenSSL build sets -mios-version-min to a very low value
// and I *think* (but I don't *know* whether) these two flags are aliasing each other.
var iosMinVersionFlagForOONIArch = map[string]string{
	"386":   "-mios-simulator-version-min=",
	"amd64": "-mios-simulator-version-min=",
	"arm":   "-miphoneos-version-min=",
	"arm64": "-miphoneos-version-min=",
}

// iosCdepsBuildArch builds the given dependency for the given arch
func iosCdepsBuildArch(deps buildtoolmodel.Dependencies, ooniArch string, name string) {
	cdenv := iosNewCBuildEnv(deps, ooniArch)
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

// iosMinVersion is the minimum version that we support.
//
// Note: "iOS 10 is the maximum deployment target for 32-bit targets".
//
// See https://stackoverflow.com/questions/47772435.
const iosMinVersion = "10.0"

// iosNewCBuildEnv creates a new [cBuildEnv] for the given ooniArch ("arm", "arm64", "386", "amd64").
func iosNewCBuildEnv(deps buildtoolmodel.Dependencies, ooniArch string) *cBuildEnv {
	destdir := runtimex.Try1(filepath.Abs(filepath.Join( // must be absolute
		"internal", "libtor", "ios", ooniArch,
	)))

	var (
		appleArch      = iosAppleArchForOONIArch[ooniArch]
		minVersionFlag = iosMinVersionFlagForOONIArch[ooniArch]
		platform       = iosPlatformForOONIArch[ooniArch]
	)
	runtimex.Assert(appleArch != "", "empty appleArch")
	runtimex.Assert(minVersionFlag != "", "empty minVersionFlag")
	runtimex.Assert(platform != "", "empty platform")

	isysroot := deps.XCRun("-sdk", platform, "--show-sdk-path")

	out := &cBuildEnv{
		ANDROID_HOME:     "", // not needed
		ANDROID_NDK_ROOT: "", // not needed
		AS:               deps.XCRun("-find", "-sdk", platform, "as"),
		AR:               deps.XCRun("-find", "-sdk", platform, "ar"),
		BINPATH:          "", // not needed
		CC:               deps.XCRun("-find", "-sdk", platform, "cc"),
		CFLAGS: []string{
			"-isysroot", isysroot,
			minVersionFlag + iosMinVersion, // tricky: they must be concatenated
			"-O2",
			"-arch", appleArch,
			"-fembed-bitcode",
		},
		CONFIGURE_HOST: "", // later
		DESTDIR:        destdir,
		CXX:            deps.XCRun("-find", "-sdk", platform, "c++"),
		CXXFLAGS: []string{
			"-isysroot", isysroot,
			minVersionFlag + iosMinVersion, // tricky: they must be concatenated
			"-arch", appleArch,
			"-fembed-bitcode",
			"-O2",
		},
		GOARCH: ooniArch,
		GOARM:  "", // maybe later
		LD:     deps.XCRun("-find", "-sdk", platform, "ld"),
		LDFLAGS: []string{
			"-isysroot", isysroot,
			minVersionFlag + iosMinVersion, // tricky: they must be concatenated
			"-arch", appleArch,
			"-fembed-bitcode",
		},
		OPENSSL_COMPILER: "", // later
		OPENSSL_POST_COMPILER_FLAGS: []string{
			minVersionFlag + iosMinVersion, // tricky: they must be concatenated
			"-fembed-bitcode",
		},
		RANLIB: deps.XCRun("-find", "-sdk", platform, "ranlib"),
		STRIP:  deps.XCRun("-find", "-sdk", platform, "strip"),
	}

	switch ooniArch {
	case "arm":
		out.CONFIGURE_HOST = "arm-apple-darwin"
		out.GOARM = "7"
		out.OPENSSL_COMPILER = "ios-xcrun"
	case "arm64":
		out.CONFIGURE_HOST = "arm-apple-darwin"
		out.GOARM = ""
		out.OPENSSL_COMPILER = "ios64-xcrun"
	case "386":
		out.CONFIGURE_HOST = "i386-apple-darwin"
		out.GOARM = ""
		out.OPENSSL_COMPILER = "iossimulator-i386-xcrun"
	case "amd64":
		out.CONFIGURE_HOST = "x86_64-apple-darwin"
		out.GOARM = ""
		out.OPENSSL_COMPILER = "iossimulator-xcrun"
	default:
		panic(errors.New("unsupported ooniArch"))
	}

	return out
}

// iosXCRun invokes `xcrun [args]` and returns its result of panics. This function
// is called indirectly by the iOS build through [buildtoolmodel.Dependencies].
func iosXCRun(args ...string) string {
	return string(must.FirstLineBytes(must.RunOutput(log.Log, "xcrun", args...)))
}
