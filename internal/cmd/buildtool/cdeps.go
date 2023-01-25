package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// cdepsEnv contains the environment for compiling a C dependency.
type cdepsEnv struct {
	// cflags contains the CFLAGS to use when compiling.
	cflags []string

	// configureHost is the value to pass to ./configure's --host option.
	configureHost string

	// destdir is the directory where to install.
	destdir string

	// lfdlags contains the LDFLAGS to use when compiling.
	ldflags []string

	// openSSLAPIDefine is an extra define we need to add on Android.
	openSSLAPIDefine string

	// openSSLCompiler is the compiler name for OpenSSL.
	openSSLCompiler string
}

// addCflags merges this struct's cflags with the extra cflags and
// then stores the merged cflags into the given envp.
func (c *cdepsEnv) addCflags(envp *shellx.Envp, extraCflags ...string) {
	mergedCflags := append([]string{}, c.cflags...)
	mergedCflags = append(mergedCflags, extraCflags...)
	envp.Append("CFLAGS", strings.Join(mergedCflags, " "))
}

// addLdflags merges this struct's ldflags with the extra ldflags and
// then stores the merged ldflags into the given envp.
func (c *cdepsEnv) addLdflags(envp *shellx.Envp, extraLdflags ...string) {
	mergedLdflags := append([]string{}, c.ldflags...)
	mergedLdflags = append(mergedLdflags, extraLdflags...)
	envp.Append("LDFLAGS", strings.Join(mergedLdflags, " "))
}

// cdepsMustMkdirTemp creates a temporary directory.
func cdepsMustMkdirTemp() string {
	return runtimex.Try1(os.MkdirTemp("", ""))
}

// cdepsMustChdir changes the current directory to the given dir and
// returns a function to return to the original working dir.
func cdepsMustChdir(work string) func() {
	prevdir := runtimex.Try1(os.Getwd())
	log.Infof("cd %s", work)
	runtimex.Try0(os.Chdir(work))
	return func() {
		runtimex.Try0(os.Chdir(prevdir))
		log.Infof("cd %s", prevdir)
	}
}

// cdepsMustFetch fetches the given URL using curl.
func cdepsMustFetch(URL string) {
	must.Run(log.Log, "curl", "-fsSLO", URL)
}

// cdepsMustVerifySHA256 verifies the SHA256 of the given tarball.
func cdepsMustVerifySHA256(expectedSHA256, tarball string) {
	firstline := string(must.FirstLineBytes(must.RunOutput(
		log.Log, "sha256sum", tarball,
	)))
	sha256, _, good := strings.Cut(firstline, " ")
	runtimex.Assert(good, "cannot obtain the first token")
	runtimex.Assert(expectedSHA256 == sha256, "SHA256 mismatch")
}

// cdepsMustAbsoluteCurdir returns the absolute path of the current dir.
func cdepsMustAbsoluteCurdir() string {
	return runtimex.Try1(filepath.Abs("."))
}

// cdepsMustListPatches returns all the patches inside a dir.
func cdepsMustListPatches(dir string) (out []string) {
	entries := runtimex.Try1(os.ReadDir(dir))
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".patch") {
			continue
		}
		out = append(out, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(out)
	return
}

// cdepsDefaultShellxConfig returns the default config used when calling shellx.RunEx.
func cdepsDefaultShellxConfig() *shellx.Config {
	return &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}
}

// cdepsMustRunWithDefaultConfig is a convenience wrapper
// around calling [shellx.RunEx] and checking the return value.
func cdepsMustRunWithDefaultConfig(envp *shellx.Envp, command string, args ...string) {
	argv := runtimex.Try1(shellx.NewArgv(command, args...))
	runtimex.Try0(shellx.RunEx(cdepsDefaultShellxConfig(), argv, envp))
}
