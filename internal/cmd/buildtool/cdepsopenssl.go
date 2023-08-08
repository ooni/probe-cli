package main

//
// Building C dependencies: OpenSSL
//
// Adapted from https://github.com/guardianproject/tor-android
// SPDX-License-Identifier: BSD-3-Clause
//

import (
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cdepsOpenSSLBuildMain is the script that builds OpenSSL.
func cdepsOpenSSLBuildMain(globalEnv *cBuildEnv, deps buildtoolmodel.Dependencies) {
	topdir := deps.AbsoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/openssl@3.rb
	cdepsMustFetch("https://www.openssl.org/source/openssl-3.1.2.tar.gz")
	deps.VerifySHA256( // must be mockable
		"a0ce69b8b97ea6a35b96875235aa453b966ba3cba8af2de23657d8b6767d6539",
		"openssl-3.1.2.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "openssl-3.1.2.tar.gz")
	_ = deps.MustChdir("openssl-3.1.2") // must be mockable

	mydir := filepath.Join(topdir, "CDEPS", "openssl")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	localEnv := &cBuildEnv{
		CFLAGS:   []string{"-Wno-macro-redefined"},
		CXXFLAGS: []string{"-Wno-macro-redefined"},
	}
	mergedEnv := cBuildMerge(globalEnv, localEnv)
	envp := cBuildExportOpenSSL(mergedEnv)

	// QUIRK: OpenSSL-3.1.2 wants ANDROID_NDK_HOME
	// TODO(bassosimone): 	Do we still need this? It seems to work without,
	//						but I can't find reliable information on this.
	if mergedEnv.ANDROID_NDK_ROOT != "" {
		envp.Append("ANDROID_NDK_HOME", mergedEnv.ANDROID_NDK_ROOT)
	}

	// QUIRK: OpenSSL-3.1.2 wants the PATH to contain the
	// directory where the Android compiler lives.
	if mergedEnv.BINPATH != "" {
		envp.Append("PATH", cdepsPrependToPath(mergedEnv.BINPATH))
	}

	argv := runtimex.Try1(shellx.NewArgv(
		"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
		"no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4", "no-mdc2",
		"no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool", "no-dso",
		"no-ui-console", "no-shared", "no-unit-test", globalEnv.OPENSSL_COMPILER,
	))
	if globalEnv.OPENSSL_API_DEFINE != "" {
		argv.Append(globalEnv.OPENSSL_API_DEFINE)
	}
	argv.Append("--libdir=lib", "--prefix=/", "--openssldir=/")
	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))

	// QUIRK: we need to supply the PATH because OpenSSL's configure
	// isn't as cool as the usual GNU configure unfortunately.
	runtimex.Try0(shellx.RunEx(
		defaultShellxConfig(),
		runtimex.Try1(shellx.NewArgv(
			"make", "-j", strconv.Itoa(runtime.NumCPU()),
		)),
		envp,
	))

	must.Run(log.Log, "make", "DESTDIR="+globalEnv.DESTDIR, "install_dev")
	must.Run(log.Log, "rm", "-rf", filepath.Join(globalEnv.DESTDIR, "lib", "pkgconfig"))
}
