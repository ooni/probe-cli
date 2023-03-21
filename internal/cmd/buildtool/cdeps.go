package main

//
// Building C dependencies: common code
//

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

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

// defaultShellxConfig returns the default config used when calling shellx.RunEx.
func defaultShellxConfig() *shellx.Config {
	return &shellx.Config{
		Logger: log.Log,
		Flags:  shellx.FlagShowStdoutStderr,
	}
}

// cdepsMustRunWithDefaultConfig is a convenience wrapper
// around calling [shellx.RunEx] and checking the return value.
func cdepsMustRunWithDefaultConfig(envp *shellx.Envp, command string, args ...string) {
	argv := runtimex.Try1(shellx.NewArgv(command, args...))
	runtimex.Try0(shellx.RunEx(defaultShellxConfig(), argv, envp))
}

// cdepsPrependToPath returns the original PATH environment
// variable with the given value prepended to it.
func cdepsPrependToPath(value string) string {
	current := os.Getenv("PATH")
	switch runtime.GOOS {
	case "windows":
		// Untested right now. If you dare running the build on pure Windows
		// and discover this code doesn't work, I owe you a beer.
		return value + ";" + current
	default:
		return value + ":" + current
	}
}
