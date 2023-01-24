package main

//
// Utility functions
//

import (
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// regularFileExists returns true if the given filepath exists and is a regular file
func regularFileExists(filepath string) bool {
	stat, err := os.Stat(filepath)
	return err == nil && stat.Mode().IsRegular()
}

// splitPair takes in input a string in the form KEY=VALUE and splits it. This
// function returns an error if it cannot find the = character to split the string.
func splitPair(s string) (string, string, error) {
	v := strings.SplitN(s, "=", 2)
	if len(v) != 2 {
		return "", "", errors.New("invalid key-value pair")
	}
	return v[0], v[1], nil
}

// mustMakeMapStringString makes a map from string to string using as input a list
// of key-value pairs used to initialize the map, or panics on error
func mustMakeMapStringString(input []string) (output map[string]string) {
	output = make(map[string]string)
	for _, opt := range input {
		key, value, err := splitPair(opt)
		runtimex.PanicOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}

// mustMakeMapStringAny makes a map from string to any using as input a list
// of key-value pairs used to initialize the map, or panics on error
func mustMakeMapStringAny(input []string) (output map[string]any) {
	output = make(map[string]any)
	for _, opt := range input {
		key, value, err := splitPair(opt)
		runtimex.PanicOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}

// gethomedir returns the home directory. If optionsHome is set, then we
// return that string as the home directory. Otherwise, we use typical
// platform-specific environment variables to determine the home. In case
// of failure to determine the home dir, we return an empty string.
func gethomedir(optionsHome string) string {
	// See https://gist.github.com/miguelmota/f30a04a6d64bd52d7ab59ea8d95e54da
	if optionsHome != "" {
		return optionsHome
	}
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	if runtime.GOOS == "linux" {
		home := os.Getenv("XDG_CONFIG_HOME")
		if home != "" {
			return home
		}
		// fallthrough
	}
	return os.Getenv("HOME")
}
