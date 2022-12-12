package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// getHomeDir returns the $HOME directory.
func getHomeDir() (string, string) {
	// See https://gist.github.com/miguelmota/f30a04a6d64bd52d7ab59ea8d95e54da
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home, "ooniprobe"
	}
	if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home, "ooniprobe"
		}
		// fallthrough
	}
	return os.Getenv("HOME"), ".ooniprobe"
}

// maybeGetOONIDir returns the $HOME/.ooniprobe equivalent unless optionsHome
// is already set, in which case it just returns optionsHome.
func maybeGetOONIDir(optionsHome string) string {
	if optionsHome != "" {
		return optionsHome
	}
	homeDir, dirName := getHomeDir()
	runtimex.Assert(homeDir != "", "homeDir is empty")
	return filepath.Join(homeDir, dirName)
}
