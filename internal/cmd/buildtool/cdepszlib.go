package main

import (
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cdepsZlibBuildMain is the script that builds zlib.
func cdepsZlibBuildMain(depsEnv *cdepsEnv) {
	topdir := cdepsMustAbsoluteCurdir()
	work := cdepsMustMkdirTemp()
	restore := cdepsMustChdir(work)
	defer restore()

	// See https://github.com/Homebrew/homebrew-core/blob/master/Formula/zlib.rb
	cdepsMustFetch("https://zlib.net/zlib-1.2.13.tar.gz")
	cdepsMustVerifySHA256(
		"b3a24de97a8fdbc835b9833169501030b8977031bcb54b3b3ac13740f846ab30",
		"zlib-1.2.13.tar.gz",
	)
	must.Run(log.Log, "tar", "-xf", "zlib-1.2.13.tar.gz")
	_ = cdepsMustChdir("zlib-1.2.13")

	mydir := filepath.Join(topdir, "CDEPS", "zlib")
	for _, patch := range cdepsMustListPatches(mydir) {
		must.Run(log.Log, "git", "apply", patch)
	}

	envp := &shellx.Envp{}
	envp.Append("CHOST", depsEnv.configureHost) // zlib's configure otherwise uses Apple's libtool
	depsEnv.fillEnv(envp)
	cdepsMustRunWithDefaultConfig(envp, "./configure", "--prefix=/", "--static")

	must.Run(log.Log, "make", "-j", strconv.Itoa(runtime.NumCPU()))

	envp = &shellx.Envp{}
	envp.Append("DESTDIR", depsEnv.destdir)
	cdepsMustRunWithDefaultConfig(envp, "make", "install")

	must.Run(log.Log, "rm", "-rf", filepath.Join(depsEnv.destdir, "lib", "pkgconfig"))
	must.Run(log.Log, "rm", "-rf", filepath.Join(depsEnv.destdir, "share"))
}
