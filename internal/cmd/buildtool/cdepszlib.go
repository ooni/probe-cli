package main

//
// Building C dependencies: zlib
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
)

// cdepsZlibBuildMain is the script that builds zlib.
func cdepsZlibBuildMain(globalEnv *cBuildEnv, deps buildtoolmodel.Dependencies) {
	topdir := deps.AbsoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/z/zlib.rb
	cdepsMustFetch("https://zlib.net/zlib-1.3.2.tar.gz")
	deps.VerifySHA256( // must be mockable
		"bb329a0a2cd0274d05519d61c667c062e06990d72e125ee2dfa8de64f0119d16",
		"zlib-1.3.2.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "zlib-1.3.2.tar.gz")
	_ = deps.MustChdir("zlib-1.3.2") // must be mockable

	mydir := filepath.Join(topdir, "CDEPS", "zlib")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	envp := cBuildExportAutotools(globalEnv)
	if globalEnv.CONFIGURE_HOST != "" {
		envp.Append("CHOST", globalEnv.CONFIGURE_HOST) // zlib's configure otherwise uses Apple's libtool
	}
	cdepsMustRunWithDefaultConfig(envp, "./configure", "--prefix=/", "--static")

	must.Run(log.Log, "make", "-j", strconv.Itoa(runtime.NumCPU()))
	must.Run(log.Log, "make", "DESTDIR="+globalEnv.DESTDIR, "install")
	must.Run(log.Log, "rm", "-rf", filepath.Join(globalEnv.DESTDIR, "lib", "pkgconfig"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(globalEnv.DESTDIR, "share"))
}
