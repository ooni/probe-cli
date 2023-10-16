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
		Use:   "cdeps [zlib|openssl|libevent|tor...]",
		Short: "Cross compiles C dependencies for iOS",
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation note: perform the check here such that we can
			// run unit test for the building code from any system
			runtimex.Assert(runtime.GOOS == "darwin", "this command requires darwin")
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
	// The ooni/probe-ios app explicitly only targets amd64 and arm64. It also targets
	// as the minimum version iOS 12, while one cannot target a version of iOS > 10 when
	// building for 32-bit targets. Hence, using only 64 bit archs here is fine.
	iosCdepsBuildArch(deps, name, "iphoneos", "arm64")
	iosCdepsBuildArch(deps, name, "iphonesimulator", "arm64")
	iosCdepsBuildArch(deps, name, "iphonesimulator", "amd64")
}

// iosAppleArchForOONIArch maps the ooniArch to the corresponding apple arch
var iosAppleArchForOONIArch = map[string]string{
	"amd64": "x86_64",
	"arm64": "arm64",
}

// iosCdepsBuildArch builds the given dependency for the given arch
func iosCdepsBuildArch(deps buildtoolmodel.Dependencies, name, platform, ooniArch string) {
	cdenv := iosNewCBuildEnv(deps, platform, ooniArch)
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

// iosMinVersion is the minimum version that we support. We're using the
// same value used by the ooni/probe-ios app as of 2023-10.12.
const iosMinVersion = "12.0"

// iosNewCBuildEnv creates a new [cBuildEnv] for the given ooniArch ("arm64" or "amd64").
func iosNewCBuildEnv(deps buildtoolmodel.Dependencies, platform, ooniArch string) *cBuildEnv {
	destdir := runtimex.Try1(filepath.Abs(filepath.Join( // must be absolute
		"internal", "libtor", platform, ooniArch,
	)))

	var (
		appleArch = iosAppleArchForOONIArch[ooniArch]

		// monVersionFlag sets the correct flag for the compiler depending on whether
		// we're using the iphoneos or iphonesimulator platform.
		//
		// Note: the documentation of clang fetched on 2023-10-12 explicitly mentions that
		// ios-version-min is an alias for iphoneos-version-min. Likewise, ios-simulator-version-min
		// aliaes iphonesimulator-version-min.
		//
		// See https://clang.llvm.org/docs/ClangCommandLineReference.html#cmdoption-clang-mios-simulator-version-min
		minVersionFlag = fmt.Sprintf("-m%s-version-min=", platform)
	)
	runtimex.Assert(appleArch != "", "empty appleArch")
	runtimex.Assert(minVersionFlag != "", "empty minVersionFlag")

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
			"-arch", appleArch,
			"-fembed-bitcode",
			"-O2",
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
		GOARM:  "", // not needed
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
	case "arm64":
		// TODO(https://github.com/ooni/probe/issues/2570): using ios64-xcrun here is wrong and
		// we should instead use the simulator as discussed in the issue.
		out.CONFIGURE_HOST = "arm-apple-darwin"
		out.OPENSSL_COMPILER = "ios64-xcrun"
	case "amd64":
		out.CONFIGURE_HOST = "x86_64-apple-darwin"
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
