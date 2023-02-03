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
		Short: "Cross compiles C dependencies for Android",
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

	// From the gomobile repository:
	//
	//	Try the ANDROID_NDK_HOME variable.  This approach is deprecated, but it has
	//	the highest priority because it represents an explicit user choice.
	//
	// See https://github.com/golang/mobile/blob/8578da983/cmd/gomobile/env.go#L394
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
				androidHome,
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
	androidHome string,
	ndkDir string,
) {
	cgo := newAndroidCBuildEnv(androidHome, ndkDir, ooniArch)

	log.Infof("building %s for android/%s", product.Pkg, ooniArch)

	argv := runtimex.Try1(shellx.NewArgv("go", "build"))
	if deps.PsiphonFilesExist() {
		argv.Append("-tags", "ooni_psiphon_config,ooni_libtor")
	} else {
		argv.Append("-tags", "ooni_libtor")
	}
	argv.Append("-ldflags", "-s -w")
	argv.Append("-o", product.DestinationPath("android", ooniArch))
	argv.Append(product.Pkg)

	// For more complex use cases such as building cdeps we have dedicated
	// extracting functions (e.g., cBuildExportAutotools), but this code is
	// simple enough that it's OK to keep here without refactoring.
	envp := &shellx.Envp{}
	envp.Append("CGO_ENABLED", "1")
	envp.Append("CC", cgo.CC)
	envp.Append("CXX", cgo.CXX)
	envp.Append("GOOS", "android")
	envp.Append("GOARCH", cgo.GOARCH)
	if cgo.GOARM != "" {
		envp.Append("GOARM", cgo.GOARM)
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
func newAndroidCBuildEnv(androidHome, ndkDir, ooniArch string) *cBuildEnv {
	binpath := androidNDKBinPath(ndkDir)
	destdir := runtimex.Try1(filepath.Abs(filepath.Join( // must be absolute
		"internal", "libtor", "android", ooniArch,
	)))
	out := &cBuildEnv{
		ANDROID_HOME:       androidHome,
		ANDROID_NDK_ROOT:   ndkDir,
		AS:                 "", // later
		AR:                 filepath.Join(binpath, "llvm-ar"),
		BINPATH:            binpath,
		CC:                 "", // later
		CFLAGS:             androidCflags(ooniArch),
		CONFIGURE_HOST:     "", // later
		DESTDIR:            destdir,
		CXX:                "", // later
		CXXFLAGS:           androidCflags(ooniArch),
		GOARCH:             ooniArch,
		GOARM:              "", // maybe later
		LD:                 filepath.Join(binpath, "ld"),
		LDFLAGS:            []string{}, // empty
		OPENSSL_API_DEFINE: "-D__ANDROID_API__=21",
		OPENSSL_COMPILER:   "", // later
		RANLIB:             filepath.Join(binpath, "llvm-ranlib"),
		STRIP:              filepath.Join(binpath, "llvm-strip"),
	}
	switch ooniArch {
	case "arm":
		out.CC = filepath.Join(out.BINPATH, "armv7a-linux-androideabi21-clang")
		out.CXX = filepath.Join(out.BINPATH, "armv7a-linux-androideabi21-clang++")
		out.GOARM = "7"
		out.CONFIGURE_HOST = "arm-linux-androideabi"
		out.OPENSSL_COMPILER = "android-arm"
	case "arm64":
		out.CC = filepath.Join(out.BINPATH, "aarch64-linux-android21-clang")
		out.CXX = filepath.Join(out.BINPATH, "aarch64-linux-android21-clang++")
		out.CONFIGURE_HOST = "aarch64-linux-android"
		out.OPENSSL_COMPILER = "android-arm64"
	case "386":
		out.CC = filepath.Join(out.BINPATH, "i686-linux-android21-clang")
		out.CXX = filepath.Join(out.BINPATH, "i686-linux-android21-clang++")
		out.CONFIGURE_HOST = "i686-linux-android"
		out.OPENSSL_COMPILER = "android-x86"
	case "amd64":
		out.CC = filepath.Join(out.BINPATH, "x86_64-linux-android21-clang")
		out.CXX = filepath.Join(out.BINPATH, "x86_64-linux-android21-clang++")
		out.CONFIGURE_HOST = "x86_64-linux-android"
		out.OPENSSL_COMPILER = "android-x86_64"
	default:
		panic(errors.New("unsupported ooniArch"))
	}
	out.AS = out.CC
	return out
}

// androidCflags returns the CFLAGS to use on Android.
func androidCflags(arch string) []string {
	// As of 2023-02-02, these are the compiler flags that ndk-build
	// would use to produce a static library. To get the flags you
	// need the following filesystem structure:
	//
	//	foobar/
	//		jni/
	//			Android.mk
	//			Application.mk
	//			hello.c
	//
	// where the content of Android.mk is the following:
	//
	//	LOCAL_PATH := $(call my-dir)
	//	include $(CLEAR_VARS)
	//	LOCAL_SRC_FILES := hello.c
	//	LOCAL_MODULE    := hello
	//	include $(BUILD_STATIC_LIBRARY)
	//
	// the content of Application.mk is the following:
	//
	//	APP_ABI := all
	//
	// the content of hello.c is the following:
	//
	//	int hello(int x) { return x; }
	//
	// To see the command line flags you need to run this command:
	//
	//	$ANDROID_NDK_ROOT/ndk-build V=1
	//
	// We discarded flags that set the target and the sysroot
	// because we use the cross compiler rather than calling
	// the clang binary directly. We also discarded all warnings
	// because we don't own the code we're compiling.
	//
	// What changed compared to previous compiler flags that
	// caused segfaults is basically that now we don't try to
	// use any sanitizer, where previously we tried some.
	//
	// We also removed -DNDEBUG because tor fails to build if
	// the -DNDEBUG compiler flag is set. (Nice!)
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
			"-mthumb",
			"-Oz",
			"-DANDROID",
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
	androidHome := deps.AndroidSDKCheck()
	ndkDir := deps.AndroidNDKCheck(androidHome)
	archs := []string{"arm", "arm64", "386", "amd64"}
	for _, arch := range archs {
		androidCdepsBuildArch(deps, arch, androidHome, ndkDir, name)
	}
}

// androidCdepsBuildArch builds the given dependency for the given arch
func androidCdepsBuildArch(
	deps buildtoolmodel.Dependencies,
	arch string,
	androidHome string,
	ndkDir string,
	name string,
) {
	cdenv := newAndroidCBuildEnv(androidHome, ndkDir, arch)
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
