package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestAndroidBuildGomobile(t *testing.T) {

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// hasPsiphon indicates whether we should build with psiphon config
		hasPsiphon bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:       "with psiphon config",
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "install", "golang.org/x/mobile/cmd/gomobile@latest",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "init",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "get", "-d", "golang.org/x/mobile/cmd/gomobile",
			},
		}, {
			Env: []string{
				"ANDROID_HOME=Android/sdk",
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
			},
			Argv: []string{
				"gomobile", "bind", "-target", "android",
				"-o", "MOBILE/android/oonimkall.aar",
				"-androidapi", "21",
				"-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w",
				"./pkg/oonimkall",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "mod", "tidy",
			},
		}},
	}, {
		name:       "without psiphon config",
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"go", "install", "golang.org/x/mobile/cmd/gomobile@latest",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"gomobile", "init",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "get", "-d", "golang.org/x/mobile/cmd/gomobile",
			},
		}, {
			Env: []string{
				"ANDROID_HOME=Android/sdk",
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
			},
			Argv: []string{
				"gomobile", "bind", "-target", "android",
				"-o", "MOBILE/android/oonimkall.aar",
				"-androidapi", "21", "-ldflags", "-s -w",
				"./pkg/oonimkall",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"go", "mod", "tidy",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: testcase.hasPsiphon,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidBuildGomobile(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGOPATH:                      1,
				buildtooltest.TagAndroidNDKCheck:             1,
				buildtooltest.TagAndroidSDKCheck:             1,
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           1,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func testAndroidGetFakeBinpath() string {
	deps := &buildtooltest.DependenciesCallCounter{}
	androidHome := deps.AndroidSDKCheck()
	sdkDir := deps.AndroidNDKCheck(androidHome)
	return androidNDKBinPath(sdkDir)
}

func TestAndroidBuildCLIAll(t *testing.T) {

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// hasPsiphon indicates whether we should build with psiphon config
		hasPsiphon bool

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	fakeBinPath := testAndroidGetFakeBinpath()

	var testcases = []testspec{{
		name:       "with psiphon config",
		hasPsiphon: true,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-android-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=386",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-android-386",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=386",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-386",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=arm64",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-android-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=arm64",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-arm64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"GOOS=android",
				"GOARCH=arm",
				"GOARM=7",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/miniooni-android-arm",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"GOOS=android",
				"GOARCH=arm",
				"GOARM=7",
			},
			Argv: []string{
				"go", "build", "-tags", "ooni_psiphon_config",
				"-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-arm",
				"./cmd/ooniprobe",
			},
		}},
	}, {
		name:       "without psiphon config",
		hasPsiphon: false,
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-android-amd64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=amd64",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-amd64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=386",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-android-386",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=386",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-386",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=arm64",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-android-arm64",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"GOOS=android",
				"GOARCH=arm64",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-arm64",
				"./cmd/ooniprobe",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"GOOS=android",
				"GOARCH=arm",
				"GOARM=7",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/miniooni-android-arm",
				"./internal/cmd/miniooni",
			},
		}, {
			Env: []string{
				"CGO_ENABLED=1",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"GOOS=android",
				"GOARCH=arm",
				"GOARM=7",
			},
			Argv: []string{
				"go", "build", "-ldflags", "-s -w", "-o", "CLI/ooniprobe-android-arm",
				"./cmd/ooniprobe",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: testcase.hasPsiphon,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidBuildCLIAll(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAndroidNDKCheck:             1,
				buildtooltest.TagAndroidSDKCheck:             1,
				buildtooltest.TagGolangCheck:                 1,
				buildtooltest.TagPsiphonMaybeCopyConfigFiles: 1,
				buildtooltest.TagPsiphonFilesExist:           8,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAndroidBuildCdepsZlib(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()
	fakeBinPath := testAndroidGetFakeBinpath()

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name: "zlib",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=" + fakeBinPath + "/llvm-ar",
				"AS=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
				"LD=" + fakeBinPath + "/ld",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CHOST=arm-linux-androideabi",
			},
			Argv: []string{
				"./configure", "--prefix=/", "--static",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/share",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=" + fakeBinPath + "/llvm-ar",
				"AS=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"LD=" + fakeBinPath + "/ld",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CHOST=aarch64-linux-android",
			},
			Argv: []string{
				"./configure", "--prefix=/", "--static",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/share",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=" + fakeBinPath + "/llvm-ar",
				"AS=" + fakeBinPath + "/i686-linux-android21-clang",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
				"LD=" + fakeBinPath + "/ld",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CHOST=i686-linux-android",
			},
			Argv: []string{
				"./configure", "--prefix=/", "--static",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/share",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.2.13.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=" + fakeBinPath + "/llvm-ar",
				"AS=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"LD=" + fakeBinPath + "/ld",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CHOST=x86_64-linux-android",
			},
			Argv: []string{
				"./configure", "--prefix=/", "--static",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/share",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: false,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidCdepsBuildMain("zlib", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir:  4,
				buildtooltest.TagAndroidNDKCheck: 1,
				buildtooltest.TagAndroidSDKCheck: 1,
				buildtooltest.TagMustChdir:       4,
				buildtooltest.TagVerifySHA256:    4,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAndroidBuildCdepsOpenSSL(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()
	fakeBinPath := testAndroidGetFakeBinpath()

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name: "openssl",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/001.patch",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
				"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
				"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
				"android-arm", "-D__ANDROID_API__=21", "--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/001.patch",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
				"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
				"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
				"android-arm64", "-D__ANDROID_API__=21", "--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/001.patch",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
				"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
				"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
				"android-x86", "-D__ANDROID_API__=21", "--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-1.1.1s.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/openssl/001.patch",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
				"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
				"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
				"android-x86_64", "-D__ANDROID_API__=21", "--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"ANDROID_NDK_HOME=Android/sdk/ndk/25.1.7654321",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_HOME=Android/sdk",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -Wno-macro-redefined",
				"ANDROID_NDK_ROOT=Android/sdk/ndk/25.1.7654321",
				"PATH=" + fakeBinPath + ":" + os.Getenv("PATH"),
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/pkgconfig",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: false,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidCdepsBuildMain("openssl", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir:  4,
				buildtooltest.TagAndroidNDKCheck: 1,
				buildtooltest.TagAndroidSDKCheck: 1,
				buildtooltest.TagMustChdir:       4,
				buildtooltest.TagVerifySHA256:    4,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAndroidBuildCdepsLibevent(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()
	fakeBinPath := testAndroidGetFakeBinpath()

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name: "libevent",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"curl",
				"-fsSLO",
				"https://github.com/libevent/libevent/archive/release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/002.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"./autogen.sh",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"LD=" + fakeBinPath + "/ld",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"AR=" + fakeBinPath + "/llvm-ar",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"LDFLAGS=-L" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib",
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/arm/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/arm/include",
				),
			},
			Argv: []string{
				"./configure",
				"--host=arm-linux-androideabi",
				"--disable-libevent-regress",
				"--disable-samples",
				"--disable-shared",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib/libevent_pthreads.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl",
				"-fsSLO",
				"https://github.com/libevent/libevent/archive/release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/002.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"./autogen.sh",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"LD=" + fakeBinPath + "/ld",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"AR=" + fakeBinPath + "/llvm-ar",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"LDFLAGS=-L" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib",
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/arm64/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/arm64/include",
				),
			},
			Argv: []string{
				"./configure",
				"--host=aarch64-linux-android",
				"--disable-libevent-regress",
				"--disable-samples",
				"--disable-shared",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib/libevent_pthreads.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl",
				"-fsSLO",
				"https://github.com/libevent/libevent/archive/release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/002.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"./autogen.sh",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/i686-linux-android21-clang",
				"LD=" + fakeBinPath + "/ld",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"AR=" + fakeBinPath + "/llvm-ar",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"LDFLAGS=-L" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib",
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/386/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/386/include",
				),
			},
			Argv: []string{
				"./configure",
				"--host=i686-linux-android",
				"--disable-libevent-regress",
				"--disable-samples",
				"--disable-shared",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib/libevent_pthreads.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl",
				"-fsSLO",
				"https://github.com/libevent/libevent/archive/release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "release-2.1.12-stable.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/libevent/002.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"./autogen.sh",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"LD=" + fakeBinPath + "/ld",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"AR=" + fakeBinPath + "/llvm-ar",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"LDFLAGS=-L" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib",
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/amd64/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/android/amd64/include",
				),
			},
			Argv: []string{
				"./configure",
				"--host=x86_64-linux-android",
				"--disable-libevent-regress",
				"--disable-samples",
				"--disable-shared",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib/libevent_pthreads.la",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: false,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidCdepsBuildMain("libevent", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir:  4,
				buildtooltest.TagAndroidNDKCheck: 1,
				buildtooltest.TagAndroidSDKCheck: 1,
				buildtooltest.TagMustChdir:       4,
				buildtooltest.TagVerifySHA256:    4,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAndroidBuildCdepsTor(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()
	fakeBinPath := testAndroidGetFakeBinpath()

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name: "tor",
		expect: []buildtooltest.ExecExpectations{{
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/002.patch",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"CC=" + fakeBinPath + "/armv7a-linux-androideabi21-clang",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -Oz -DANDROID -fsanitize=bounds -fsanitize-undefined-trap-on-error -mthumb",
				"CXX=" + fakeBinPath + "/armv7a-linux-androideabi21-clang++",
				"LD=" + fakeBinPath + "/ld",
				"AR=" + fakeBinPath + "/llvm-ar",
			},
			Argv: []string{
				"./configure",
				"--host=arm-linux-androideabi",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "src/feature/api/tor_api.h",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm/lib",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/002.patch",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"CC=" + fakeBinPath + "/aarch64-linux-android21-clang",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fpic -O2 -DANDROID -fsanitize=safe-stack -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CXX=" + fakeBinPath + "/aarch64-linux-android21-clang++",
				"LD=" + fakeBinPath + "/ld",
				"AR=" + fakeBinPath + "/llvm-ar",
			},
			Argv: []string{
				"./configure",
				"--host=aarch64-linux-android",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "src/feature/api/tor_api.h",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/arm64/lib",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/002.patch",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/i686-linux-android21-clang",
				"CC=" + fakeBinPath + "/i686-linux-android21-clang",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error -mstackrealign",
				"CXX=" + fakeBinPath + "/i686-linux-android21-clang++",
				"LD=" + fakeBinPath + "/ld",
				"AR=" + fakeBinPath + "/llvm-ar",
			},
			Argv: []string{
				"./configure",
				"--host=i686-linux-android",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "src/feature/api/tor_api.h",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/386/lib",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.7.12.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/000.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/001.patch",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/002.patch",
			},
		}, {
			Env: []string{
				"AS=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"CC=" + fakeBinPath + "/x86_64-linux-android21-clang",
				"RANLIB=" + fakeBinPath + "/llvm-ranlib",
				"STRIP=" + fakeBinPath + "/llvm-strip",
				"CXXFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CFLAGS=-fdata-sections -ffunction-sections -fstack-protector-strong -funwind-tables -no-canonical-prefixes -D_FORTIFY_SOURCE=2 -fPIC -O2 -DANDROID -fsanitize=safe-stack -fstack-clash-protection -fsanitize=bounds -fsanitize-undefined-trap-on-error",
				"CXX=" + fakeBinPath + "/x86_64-linux-android21-clang++",
				"LD=" + fakeBinPath + "/ld",
				"AR=" + fakeBinPath + "/llvm-ar",
			},
			Argv: []string{
				"./configure",
				"--host=x86_64-linux-android",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "src/feature/api/tor_api.h",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/android/amd64/lib",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {

			cc := &buildtooltest.SimpleCommandCollector{}

			deps := &buildtooltest.DependenciesCallCounter{
				HasPsiphon: false,
			}

			shellxtesting.WithCustomLibrary(cc, func() {
				androidCdepsBuildMain("tor", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir:  4,
				buildtooltest.TagAndroidNDKCheck: 1,
				buildtooltest.TagAndroidSDKCheck: 1,
				buildtooltest.TagMustChdir:       4,
				buildtooltest.TagVerifySHA256:    4,
			}

			if diff := cmp.Diff(expectCalls, deps.Counter); diff != "" {
				t.Fatal(diff)
			}

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}
