package main

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtooltest"
	"github.com/ooni/probe-cli/v3/internal/shellx/shellxtesting"
)

func TestLinuxCdepsBuildMain(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOARCH != "amd64" {
		t.Skip("skip test for GOOS != linux and GOARCH != amd64")
	}

	faketopdir := (&buildtooltest.DependenciesCallCounter{}).AbsoluteCurDir()

	// testspec specifies a test case for this test
	type testspec struct {
		// name is the name of the test case
		name string

		// target is the target to build
		target string

		// expectations contains the commands we expect to see
		expect []buildtooltest.ExecExpectations
	}

	var testcases = []testspec{{
		name:   "we can build zlib",
		target: "zlib",
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
				"CFLAGS=-D_FORTIFY_SOURCE=2 -fstack-protector-strong -fstack-clash-protection -fPIC -fsanitize=bounds -fsanitize-undefined-trap-on-error -O2",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/share",
			},
		}},
	}, {
		name:   "we can build openssl",
		target: "openssl",
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
				"CFLAGS=-D_FORTIFY_SOURCE=2 -fstack-protector-strong -fstack-clash-protection -fPIC -fsanitize=bounds -fsanitize-undefined-trap-on-error -O2 -Wno-macro-redefined",
			},
			Argv: []string{
				"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
				"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
				"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
				"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
				"linux-x86_64", "--libdir=lib", "--prefix=/", "--openssldir=/",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
				"install_dev",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm", "-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/pkgconfig",
			},
		}},
	}, {
		name:   "we can build libevent",
		target: "libevent",
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
				fmt.Sprintf(
					"CFLAGS=-D_FORTIFY_SOURCE=2 -fstack-protector-strong -fstack-clash-protection -fPIC -fsanitize=bounds -fsanitize-undefined-trap-on-error -O2 -I%s/internal/cmd/buildtool/internal/libtor/linux/amd64/include",
					faketopdir,
				),
				fmt.Sprintf(
					"LDFLAGS=-L%s/internal/cmd/buildtool/internal/libtor/linux/amd64/lib",
					faketopdir,
				),
			},
			Argv: []string{
				"./configure",
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
				"DESTDIR=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
				"install",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/bin",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/pkgconfig",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_core.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_core.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_extra.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_extra.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_openssl.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_openssl.la",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_pthreads.a",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"rm",
				"-rf",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib/libevent_pthreads.la",
			},
		}},
	}, {
		name:   "we can build tor",
		target: "tor",
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
				"CFLAGS=-D_FORTIFY_SOURCE=2 -fstack-protector-strong -fstack-clash-protection -fPIC -fsanitize=bounds -fsanitize-undefined-trap-on-error -O2",
				"LDFLAGS=",
			},
			Argv: []string{
				"./configure",
				"--enable-pic",
				"--enable-static-libevent",
				"--with-libevent-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
				"--enable-static-openssl",
				"--with-openssl-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
				"--enable-static-zlib",
				"--with-zlib-dir=" + faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64",
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
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/include",
			},
		}, {
			Env: []string{},
			Argv: []string{
				"install", "-m644", "libtor.a",
				faketopdir + "/internal/cmd/buildtool/internal/libtor/linux/amd64/lib",
			},
		}},
	}}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			cc := &buildtooltest.SimpleCommandCollector{}

			shellxtesting.WithCustomLibrary(cc, func() {
				linuxCdepsBuildMain(testcase.target, &buildtooltest.DependenciesCallCounter{})
			})

			if err := buildtooltest.CheckManyCommands(cc.Commands, testcase.expect); err != nil {
				t.Fatal(err)
			}
		})
	}
}
