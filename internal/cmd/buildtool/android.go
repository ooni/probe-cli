package main

//
// Android build
//

import (
	"errors"
	"fmt"
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
		Short: "Builds ooniprobe, miniooni, and oonimkall for android",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "gomobile",
		Short: "Builds oonimkall for android using gomobile",
		Run: func(cmd *cobra.Command, args []string) {
			androidBuildGomobile(&buildDeps{})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "cli",
		Short: "Builds ooniprobe and miniooni for usage within termux",
		Run: func(cmd *cobra.Command, args []string) {
			androidBuildCLIAll(&buildDeps{})
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "cdeps {zlib|openssl|libevent|tor} [zlib|openssl|libevent|tor...]",
		Short: "Builds C dependencies on Linux systems (experimental)",
		Run: func(cmd *cobra.Command, args []string) {
			for _, arg := range args {
				androidCdepsBuildMain(arg, &buildDeps{})
			}
		},
		Args: cobra.MinimumNArgs(1),
	})
	return cmd
}

// androidBuildGomobile invokes the gomobile build.
func androidBuildGomobile(deps buildtoolmodel.Dependencies) {
	runtimex.Assert(
		runtime.GOOS == "darwin" || runtime.GOOS == "linux",
		"this command requires darwin or linux",
	)

	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	androidHome := deps.AndroidSDKCheck()
	ndkDir := deps.AndroidNDKCheck(androidHome)

	envp := &shellx.Envp{}
	envp.Append("ANDROID_HOME", androidHome)
	envp.Append("ANDROID_NDK_HOME", ndkDir)

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

// androidBuildCLIAll builds all products in CLI mode for Android
func androidBuildCLIAll(deps buildtoolmodel.Dependencies) {
	runtimex.Assert(
		runtime.GOOS == "darwin" || runtime.GOOS == "linux",
		"this command requires darwin or linux",
	)

	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	androidHome := deps.AndroidSDKCheck()
	ndkDir := deps.AndroidNDKCheck(androidHome)
	archs := []string{"amd64", "386", "arm64", "arm"}
	products := []*product{productMiniooni, productOoniprobe}
	for _, arch := range archs {
		for _, product := range products {
			androidBuildCLIProductArch(
				deps,
				product,
				arch,
				ndkDir,
			)
		}
	}
}

// androidBuildCLIProductArch builds a product for the given arch.
func androidBuildCLIProductArch(
	deps buildtoolmodel.Dependencies,
	product *product,
	ooniArch string,
	ndkDir string,
) {
	cdeps := newAndroidCBuildEnv(ndkDir, ooniArch)

	log.Infof("building %s for android/%s", product.Pkg, ooniArch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append("-o", product.DestinationPath("android", ooniArch))
	argv.Append(product.Pkg)

	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "1")
	envp.Append("CC", cdeps.cc)
	envp.Append("CXX", cdeps.cxx)
	envp.Append("GOOS", "android")
	envp.Append("GOARCH", cdeps.goarch)
	if cdeps.goarm != "" {
		envp.Append("GOARM", cdeps.goarm)
	}

	// [2023-01-26] Adding the following flags produces these warnings for android/arm
	//
	//	ld: warning: /tmp/go-link-2920159630/000016.o:(function threadentry: .text.threadentry+0x16):
	//	branch and link relocation: R_ARM_THM_CALL to non STT_FUNC symbol: crosscall_arm1 interworking
	//	not performed; consider using directive '.type crosscall_arm1, %function' to give symbol
	//	type STT_FUNC if interworking between ARM and Thumb is required; gcc_linux_arm.c
	//
	// So, for now, I have disabled adding the flags.
	//
	//envp.Append("CGO_CFLAGS", strings.Join(cgo.cflags, " "))
	//envp.Append("CGO_CXXFLAGS", strings.Join(cgo.cxxflags, " "))

	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))
}

// newAndroidCBuildEnv creates a new [cBuildEnv] for the
// given ooniArch ("arm", "arm64", "386", "amd64").
func newAndroidCBuildEnv(ndkDir string, ooniArch string) *cBuildEnv {
	out := &cBuildEnv{
		binpath:          androidNDKBinPath(ndkDir),
		cc:               "",
		cflags:           androidCflags(ooniArch),
		configureHost:    "",
		destdir:          "",
		cxx:              "",
		cxxflags:         androidCflags(ooniArch),
		goarch:           "",
		goarm:            "",
		ldflags:          []string{},
		openSSLAPIDefine: "",
		openSSLCompiler:  "",
	}
	switch ooniArch {
	case "arm":
		out.cc = filepath.Join(out.binpath, "armv7a-linux-androideabi21-clang")
		out.cxx = filepath.Join(out.binpath, "armv7a-linux-androideabi21-clang++")
		out.goarch = ooniArch
		out.goarm = "7"
		out.openSSLCompiler = "android-arm"
	case "arm64":
		out.cc = filepath.Join(out.binpath, "aarch64-linux-android21-clang")
		out.cxx = filepath.Join(out.binpath, "aarch64-linux-android21-clang++")
		out.goarch = ooniArch
		out.goarm = ""
		out.openSSLCompiler = "android-arm64"
	case "386":
		out.cc = filepath.Join(out.binpath, "i686-linux-android21-clang")
		out.cxx = filepath.Join(out.binpath, "i686-linux-android21-clang++")
		out.goarch = ooniArch
		out.goarm = ""
		out.openSSLCompiler = "android-x86"
	case "amd64":
		out.cc = filepath.Join(out.binpath, "x86_64-linux-android21-clang")
		out.cxx = filepath.Join(out.binpath, "x86_64-linux-android21-clang++")
		out.goarch = ooniArch
		out.goarm = ""
		out.openSSLCompiler = "android-x86_64"
	default:
		panic(errors.New("unsupported ooniArch"))
	}
	return out
}

// androidCflags returns the CFLAGS to use on Android.
func androidCflags(arch string) []string {
	// See https://airbus-seclab.github.io/c-compiler-security/ as well as the flags
	// produced by running ndk-build inside the android/ndk-samples repository
	// (see https://github.com/android/ndk-samples/tree/android-mk/hello-jni/jni).
	//
	// TODO(bassosimone): as of 2023-01-10, -fstack-clash-protection causes
	// a warning when compiling for either arm or arm64.
	//
	// TODO(bassosimone): as of 2023-01-10, -fsanitize=safe-stack is not
	// defined when compiling for arm and causes a linker error. (It's curious
	// that we see a linker error but this happens because zlib also builds
	// some examples as part of its default build.)
	switch arch {
	case "386":
		return []string{
			"-fdata-sections",
			"-ffunction-sections",
			"-fstack-protector-strong",
			"-funwind-tables",
			"-no-canonical-prefixes",
			"-D_FORTIFY_SOURCE=2",
			"-fPIC",
			"-O2",
			"-DANDROID",
			"-fsanitize=safe-stack",
			"-fstack-clash-protection",
			"-fsanitize=bounds",
			"-fsanitize-undefined-trap-on-error",
			"-mstackrealign",
		}
	case "amd64":
		return []string{
			"-fdata-sections",
			"-ffunction-sections",
			"-fstack-protector-strong",
			"-funwind-tables",
			"-no-canonical-prefixes",
			"-D_FORTIFY_SOURCE=2",
			"-fPIC",
			"-O2",
			"-DANDROID",
			"-fsanitize=safe-stack",
			"-fstack-clash-protection",
			"-fsanitize=bounds",
			"-fsanitize-undefined-trap-on-error",
		}
	case "arm":
		return []string{
			"-fdata-sections",
			"-ffunction-sections",
			"-fstack-protector-strong",
			"-funwind-tables",
			"-no-canonical-prefixes",
			"-D_FORTIFY_SOURCE=2",
			"-fpic",
			"-Oz",
			"-DANDROID",
			"-fsanitize=bounds",
			"-fsanitize-undefined-trap-on-error",
			"-mthumb",
		}
	case "arm64":
		return []string{
			"-fdata-sections",
			"-ffunction-sections",
			"-fstack-protector-strong",
			"-funwind-tables",
			"-no-canonical-prefixes",
			"-D_FORTIFY_SOURCE=2",
			"-fpic",
			"-O2",
			"-DANDROID",
			"-fsanitize=safe-stack",
			"-fsanitize=bounds",
			"-fsanitize-undefined-trap-on-error",
		}
	default:
		panic(errors.New("unsupported arch"))
	}
}

// androidNDKBinPath returns the binary path given the android
// NDK home and the runtime.GOOS variable value.
func androidNDKBinPath(ndkDir string) string {
	// TODO(bassosimone): do android toolchains exists for other runtime.GOARCH?
	switch runtime.GOOS {
	case "linux":
		return filepath.Join(ndkDir, "toolchains", "llvm", "prebuilt", "linux-x86_64", "bin")
	case "darwin":
		return filepath.Join(ndkDir, "toolchains", "llvm", "prebuilt", "darwin-x86_64", "bin")
	default:
		panic(errors.New("unsupported runtime.GOOS"))
	}
}

// androidCdepsBuildMain builds C dependencies for android.
func androidCdepsBuildMain(name string, deps buildtoolmodel.Dependencies) {
	runtimex.Assert(
		runtime.GOOS == "darwin" || runtime.GOOS == "linux",
		"this command requires darwin or linux",
	)
	deps.PsiphonMaybeCopyConfigFiles()
	deps.GolangCheck()

	androidHome := deps.AndroidSDKCheck()
	ndkDir := deps.AndroidNDKCheck(androidHome)
	//archs := []string{"amd64", "386", "arm64", "arm"}
	archs := []string{"arm64"} //XXX
	for _, arch := range archs {
		androidCdepsBuildArch(deps, arch, ndkDir, name)
	}
}

// androidCdepsBuildArch builds the given dependency for the given arch
func androidCdepsBuildArch(deps buildtoolmodel.Dependencies, arch, ndkDir, name string) {
	cdenv := newAndroidCBuildEnv(ndkDir, arch)
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
