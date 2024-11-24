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

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/o/openssl@3.rb
	cdepsMustFetch("https://www.openssl.org/source/openssl-3.4.0.tar.gz")
	deps.VerifySHA256( // must be mockable
		"e15dda82fe2fe8139dc2ac21a36d4ca01d5313c75f99f46c4e8a27709b7294bf",
		"openssl-3.4.0.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "openssl-3.4.0.tar.gz")
	_ = deps.MustChdir("openssl-3.4.0") // must be mockable

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

	// QUIRK: OpenSSL-1.1.1v wanted ANDROID_NDK_HOME
	// TODO(bassosimone): do we still need this? It seems to work without,
	// but I can't find reliable information on this.
	if mergedEnv.ANDROID_NDK_ROOT != "" {
		envp.Append("ANDROID_NDK_HOME", mergedEnv.ANDROID_NDK_ROOT)
	}

	// QUIRK: OpenSSL-1.1.1v wanted the PATH to contain the
	// directory where the Android compiler lives.
	// TODO(bassosimone): do we still need this?
	if mergedEnv.BINPATH != "" {
		envp.Append("PATH", cdepsPrependToPath(mergedEnv.BINPATH))
	}

	argv := runtimex.Try1(shellx.NewArgv(
		"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
		"no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4", "no-mdc2",
		"no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool", "no-dso",
		"no-ui-console", "no-shared", "no-unit-test", globalEnv.OPENSSL_COMPILER,
	))
	argv.Append(globalEnv.OPENSSL_POST_COMPILER_FLAGS...)
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

	// We used to delete the pkgconfig but it turns out this is important for libevent iOS builds, which
	// means now we need to keep it. See https://github.com/ooni/probe-cli/pull/1369 for details.
	//must.Run(log.Log, "rm", "-rf", filepath.Join(globalEnv.DESTDIR, "lib", "pkgconfig"))
}
