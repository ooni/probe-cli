package main

//
// Building C dependencies: libevent
//
// Adapted from https://github.com/guardianproject/tor-android
// SPDX-License-Identifier: BSD-3-Clause

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

// cdepsLibeventBuildMain is the script that builds libevent.
func cdepsLibeventBuildMain(cdenv *cdepsEnv, deps buildtoolmodel.Dependencies) {
	topdir := deps.AbsoluteCurDir() // must be mockable
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/libevent.rb
	cdepsMustFetch("https://github.com/libevent/libevent/archive/release-2.1.12-stable.tar.gz")
	deps.VerifySHA256( // must be mockable
		"7180a979aaa7000e1264da484f712d403fcf7679b1e9212c4e3d09f5c93efc24",
		"release-2.1.12-stable.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "release-2.1.12-stable.tar.gz")
	_ = deps.MustChdir("libevent-release-2.1.12-stable") // must be mockable

	mydir := filepath.Join(topdir, "CDEPS", "libevent")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	must.Run(log.Log, "./autogen.sh")

	envp := &shellx.Envp{}
	cdepsAddCflags(envp, cdenv, "-I"+cdenv.destdir+"/include")
	cdepsAddLdflags(envp, cdenv, "-L"+cdenv.destdir+"/lib")

	argv := runtimex.Try1(shellx.NewArgv("./configure"))
	if cdenv.configureHost != "" {
		argv.Append("--host=" + cdenv.configureHost)
	}
	argv.Append("--disable-libevent-regress", "--disable-samples", "--disable-shared", "--prefix=/")
	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))

	must.Run(log.Log, "make", "V=1", "-j", strconv.Itoa(runtime.NumCPU()))
	must.Run(log.Log, "make", "DESTDIR="+cdenv.destdir, "install")
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "bin"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "pkgconfig"))

	// we just need libevent.a
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent.la"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_core.a"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_core.la"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_extra.a"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_extra.la"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_openssl.a"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_openssl.la"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_pthreads.a"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(cdenv.destdir, "lib", "libevent_pthreads.la"))
}
