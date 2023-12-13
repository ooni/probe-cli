package main

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestIOSBuildGomobile(t *testing.T) {

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
			Env: []string{},
			Argv: []string{
				"gomobile", "bind", "-target", "ios",
				"-o", "MOBILE/ios/oonimkall.xcframework",
				"-tags", "ooni_psiphon_config,ooni_libtor",
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
			Env: []string{},
			Argv: []string{
				"gomobile", "bind", "-target", "ios",
				"-o", "MOBILE/ios/oonimkall.xcframework",
				"-tags", "ooni_libtor",
				"-ldflags", "-s -w", "./pkg/oonimkall",
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
				iosBuildGomobile(deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagGOPATH:                      1,
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

func TestIOSBuildCdepsZlib(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

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
				"curl", "-fsSLO", "https://zlib.net/zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=/Developer/SDKs/iphoneos/bin/ar",
				"AS=/Developer/SDKs/iphoneos/bin/as",
				"CC=/Developer/SDKs/iphoneos/bin/cc",
				"CFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"CXX=/Developer/SDKs/iphoneos/bin/c++",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"LD=/Developer/SDKs/iphoneos/bin/ld",
				"LDFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode",
				"RANLIB=/Developer/SDKs/iphoneos/bin/ranlib",
				"STRIP=/Developer/SDKs/iphoneos/bin/strip",
				"CHOST=arm-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/share",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				"CHOST=arm-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/share",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://zlib.net/zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "zlib-1.3.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/zlib/000.patch",
			},
		}, {
			Env: []string{
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				"CHOST=x86_64-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf", faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/share",
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
				iosCdepsBuildMain("zlib", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir: 3,
				buildtooltest.TagMustChdir:      3,
				buildtooltest.TagVerifySHA256:   3,
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

func TestIOSBuildCdepsOpenSSL(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

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
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-3.2.0.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-3.2.0.tar.gz",
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
				"CFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4", "no-mdc2",
				"no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool", "no-dso",
				"no-ui-console", "no-shared", "no-unit-test", "ios64-xcrun",
				"-miphoneos-version-min=12.0", "-fembed-bitcode",
				"--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"CFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-3.2.0.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-3.2.0.tar.gz",
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
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4", "no-mdc2",
				"no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool", "no-dso",
				"no-ui-console", "no-shared", "no-unit-test", "ios64-xcrun",
				"-miphonesimulator-version-min=12.0", "-fembed-bitcode",
				"--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.openssl.org/source/openssl-3.2.0.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "openssl-3.2.0.tar.gz",
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
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4", "no-mdc2",
				"no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool", "no-dso",
				"no-ui-console", "no-shared", "no-unit-test", "iossimulator-xcrun",
				"-miphonesimulator-version-min=12.0", "-fembed-bitcode",
				"--libdir=lib", "--prefix=/", "--openssldir=/",
			},
		}, {
			Env: []string{
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2 -Wno-macro-redefined",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"make", "-j", strconv.Itoa(runtime.NumCPU()),
			},
		}, {
			Env: []string{},
			Argv: []string{
				"make",
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"install_dev",
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
				iosCdepsBuildMain("openssl", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir: 3,
				buildtooltest.TagMustChdir:      3,
				buildtooltest.TagVerifySHA256:   3,
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

func TestIOSBuildCdepsLibevent(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

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
				"AS=/Developer/SDKs/iphoneos/bin/as",
				"LD=/Developer/SDKs/iphoneos/bin/ld",
				"CXX=/Developer/SDKs/iphoneos/bin/c++",
				"CC=/Developer/SDKs/iphoneos/bin/cc",
				"AR=/Developer/SDKs/iphoneos/bin/ar",
				"RANLIB=/Developer/SDKs/iphoneos/bin/ranlib",
				"STRIP=/Developer/SDKs/iphoneos/bin/strip",
				fmt.Sprintf(
					"%s %s",
					"LDFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode",
					"-L"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib",
				),
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/include",
				),
				"PKG_CONFIG_PATH=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig",
			},
			Argv: []string{
				"./configure",
				"--host=arm-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig/libevent.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig/libevent_core.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig/libevent_extra.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig/libevent_openssl.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/pkgconfig/libevent_pthreads.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib/libevent_pthreads.la",
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
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				fmt.Sprintf(
					"%s %s",
					"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode",
					"-L"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib",
				),
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/include",
				),
				"PKG_CONFIG_PATH=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig",
			},
			Argv: []string{
				"./configure",
				"--host=arm-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig/libevent.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig/libevent_core.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig/libevent_extra.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig/libevent_openssl.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/pkgconfig/libevent_pthreads.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib/libevent_pthreads.la",
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
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				fmt.Sprintf(
					"%s %s",
					"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode",
					"-L"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib",
				),
				fmt.Sprintf(
					"%s %s",
					"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/include",
				),
				fmt.Sprintf(
					"%s %s",
					"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
					"-I"+faketopdir+"/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/include",
				),
				"PKG_CONFIG_PATH=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig",
			},
			Argv: []string{
				"./configure",
				"--host=x86_64-apple-darwin",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig/libevent.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig/libevent_core.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig/libevent_extra.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig/libevent_openssl.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-f",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/pkgconfig/libevent_pthreads.pc",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib/libevent_pthreads.la",
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
				iosCdepsBuildMain("libevent", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir: 3,
				buildtooltest.TagMustChdir:      3,
				buildtooltest.TagVerifySHA256:   3,
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

func TestIOSBuildCdepsTor(t *testing.T) {
	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

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
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.8.10.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.8.10.tar.gz",
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
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/003.patch",
			},
		}, {
			Env: []string{
				"AS=/Developer/SDKs/iphoneos/bin/as",
				"CC=/Developer/SDKs/iphoneos/bin/cc",
				"RANLIB=/Developer/SDKs/iphoneos/bin/ranlib",
				"STRIP=/Developer/SDKs/iphoneos/bin/strip",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"CFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"LDFLAGS=-isysroot /Developer/SDKs/iphoneos -miphoneos-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXX=/Developer/SDKs/iphoneos/bin/c++",
				"LD=/Developer/SDKs/iphoneos/bin/ld",
				"AR=/Developer/SDKs/iphoneos/bin/ar",
			},
			Argv: []string{
				"./configure",
				"--host=arm-apple-darwin",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
				"--disable-unittests",
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
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphoneos/arm64/lib",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.8.10.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.8.10.tar.gz",
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
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/003.patch",
			},
		}, {
			Env: []string{
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode -O2",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch arm64 -fembed-bitcode",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
			},
			Argv: []string{
				"./configure",
				"--host=arm-apple-darwin",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
				"--disable-unittests",
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
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/arm64/lib",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"curl", "-fsSLO", "https://www.torproject.org/dist/tor-0.4.8.10.tar.gz",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"tar", "-xf", "tor-0.4.8.10.tar.gz",
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
			Env: []string{},
			Argv: []string{
				"git", "apply", faketopdir + "/CDEPS/tor/003.patch",
			},
		}, {
			Env: []string{
				"AS=/Developer/SDKs/iphonesimulator/bin/as",
				"CC=/Developer/SDKs/iphonesimulator/bin/cc",
				"RANLIB=/Developer/SDKs/iphonesimulator/bin/ranlib",
				"STRIP=/Developer/SDKs/iphonesimulator/bin/strip",
				"CXXFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
				"CFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode -O2",
				"LDFLAGS=-isysroot /Developer/SDKs/iphonesimulator -miphonesimulator-version-min=12.0 -arch x86_64 -fembed-bitcode",
				"CXX=/Developer/SDKs/iphonesimulator/bin/c++",
				"LD=/Developer/SDKs/iphonesimulator/bin/ld",
				"AR=/Developer/SDKs/iphonesimulator/bin/ar",
			},
			Argv: []string{
				"./configure",
				"--host=x86_64-apple-darwin",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64",
				"--disable-module-dirauth",
				"--disable-zstd",
				"--disable-lzma",
				"--disable-tool-name-check",
				"--disable-systemd",
				"--prefix=/",
				"--disable-unittests",
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
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/iphonesimulator/amd64/lib",
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
				iosCdepsBuildMain("tor", deps)
			})

			expectCalls := map[string]int{
				buildtooltest.TagAbsoluteCurDir: 3,
				buildtooltest.TagMustChdir:      3,
				buildtooltest.TagVerifySHA256:   3,
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
