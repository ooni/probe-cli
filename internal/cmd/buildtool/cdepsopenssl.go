package main

// Adapted from https://github.com/guardianproject/tor-android
// SPDX-License-Identifier: BSD-3-Clause

import (
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cdepsOpenSSLBuildMain is the script that builds OpenSSL.
func cdepsOpenSSLBuildMain(depsEnv *cdepsEnv) {
	topdir := cdepsMustAbsoluteCurdir()
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/openssl@1.1.rb
	cdepsMustFetch("https://www.openssl.org/source/openssl-1.1.1s.tar.gz")
	cdepsMustVerifySHA256(
		"c5ac01e760ee6ff0dab61d6b2bbd30146724d063eb322180c6f18a6f74e4b6aa",
		"openssl-1.1.1s.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "openssl-1.1.1s.tar.gz")
	_ = cdepsMustChdir("openssl-1.1.1s")

	mydir := filepath.Join(topdir, "CDEPS", "openssl")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	envp := &shellx.Envp{}
	depsEnv.addCflags(envp, "-Wno-macro-redefined")
	argv := runtimex.Try1(shellx.NewArgv(
		"./Configure", "no-comp", "no-dtls", "no-ec2m", "no-psk", "no-srp",
		"no-ssl2", "no-ssl3", "no-camellia", "no-idea", "no-md2", "no-md4",
		"no-mdc2", "no-rc2", "no-rc4", "no-rc5", "no-rmd160", "no-whirlpool",
		"no-dso", "no-hw", "no-ui-console", "no-shared", "no-unit-test",
		depsEnv.openSSLCompiler,
	))
	if depsEnv.openSSLAPIDefine != "" {
		argv.Append(depsEnv.openSSLAPIDefine)
	}
	argv.Append("--libdir=lib", "--prefix=/", "--openssldir=/")
	runtimex.Try0(shellx.RunEx(cdepsDefaultShellxConfig(), argv, envp))

	must.Run(log.Log, "make", "-j", strconv.Itoa(runtime.NumCPU()))
	must.Run(log.Log, "make", "DESTDIR="+depsEnv.destdir, "install_dev")
	must.Run(log.Log, "rm", "-rf", filepath.Join(depsEnv.destdir, "lib", "pkgconfig"))
}
