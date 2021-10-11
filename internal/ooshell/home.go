package ooshell

//
// home.go
//
// Code to detect $HOME
//

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrCannotDetermineHomeDir means we cannot determine the $HOME dir.
var ErrCannotDetermineHomeDir = errors.New("cannot determine the $HOME dir")

// OONIHome returns the location of the OONI_HOME directory by
// combining the HOME directory with the given dirName.
func OONIHome(dirName string) (string, error) {
	home, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, dirName), nil
}

// Home returns the home directory or an error.
func Home() (string, error) {
	if hd := home(); hd != "" {
		return hd, nil
	}
	return "", ErrCannotDetermineHomeDir
}

// home implements HOME.
func home() string {
	// See https://gist.github.com/miguelmota/f30a04a6d64bd52d7ab59ea8d95e54da
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
