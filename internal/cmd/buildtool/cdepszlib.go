package main

// Adapted from https://github.com/guardianproject/tor-android
// SPDX-License-Identifier: BSD-3-Clause

import (
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cdepsZlibBuildMain is the script that builds zlib.
func cdepsZlibBuildMain(cdenv *cdepsEnv, deps cdepsDependencies) {
	topdir := deps.absoluteCurDir()
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/zlib.rb
	cdepsMustFetch("https://zlib.net/zlib-1.2.13.tar.gz")
	deps.verifySHA256(
		"b3a24de97a8fdbc835b9833169501030b8977031bcb54b3b3ac13740f846ab30",
		"zlib-1.2.13.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "zlib-1.2.13.tar.gz")
	_ = deps.mustChdir("zlib-1.2.13")

	mydir := filepath.Join(topdir, "CDEPS", "zlib")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	envp := &shellx.Envp{}
	if cdenv.configureHost != "" {
		envp.Append("CHOST", cdenv.configureHost) // zlib's configure otherwise uses Apple's libtool
	}
	cdenv.addCflags(envp)
	cdepsMustRunWithDefaultConfig(envp, "./configure", "--prefix=/", "--static")

	must.Run(log.Log, "make", "-j", strconv.Itoa(runtime.NumCPU()))
	must.Run(log.Log, "make", "DESTDIR="+cdenv.destdir, "install")
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "pkgconfig"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "share"))
}
