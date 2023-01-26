package main

//
// Building C dependencies: tor
//
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

// cdepsTorBuildMain is the script that builds tor.
func cdepsTorBuildMain(cdenv *cdepsEnv, deps cdepsDependencies) {
	topdir := deps.absoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/tor.rb
	cdepsMustFetch("https://www.torproject.org/dist/tor-0.4.7.12.tar.gz")
	deps.verifySHA256( // must be mockable
		"3b5d969712c467851bd028f314343ef15a97ea457191e93ffa97310b05b9e395",
		"tor-0.4.7.12.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "tor-0.4.7.12.tar.gz")
	_ = deps.mustChdir("tor-0.4.7.12") // must be mockable

	mydir := filepath.Join(topdir, "CDEPS", "tor")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	envp := &shellx.Envp{}
	cdepsAddCflags(envp, cdenv)
	cdepsAddLdflags(envp, cdenv)

	argv := runtimex.Try1(shellx.NewArgv("./configure"))
	if cdenv.configureHost != "" {
		argv.Append("--host=" + cdenv.configureHost)
	}
	argv.Append(
		"--enable-pic",
		"--enable-static-libevent", "--with-libevent-dir="+cdenv.destdir,
		"--enable-static-openssl", "--with-openssl-dir="+cdenv.destdir,
		"--enable-static-zlib", "--with-zlib-dir="+cdenv.destdir,
		"--disable-module-dirauth",
		"--disable-zstd", "--disable-lzma",
		"--disable-tool-name-check",
		"--disable-systemd",
		"--prefix=/",
	)
	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))

	must.Run(log.Log, "make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()))

	must.Run(log.Log, "install", "-m644", "src/feature/api/tor_api.h", cdenv.destdir+"/include")
	must.Run(log.Log, "install", "-m644", "libtor.a", cdenv.destdir+"/lib")
}
