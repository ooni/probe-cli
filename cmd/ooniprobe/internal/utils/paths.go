package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils/homedir"
	"github.com/ooni/probe-cli/v3/internal/fsx"
)

// RequiredDirs returns the required ooni home directories
func RequiredDirs(home string) []string {
	requiredDirs := []string{}
	requiredSubdirs := []string{"assets", "db", "msmts"}
	for _, d := range requiredSubdirs {
		requiredDirs = append(requiredDirs, filepath.Join(home, d))
	}
	return requiredDirs
}

// ConfigPath returns the default path to the config file
func ConfigPath(home string) string {
	return filepath.Join(home, "config.json")
}

// AssetsDir returns the assets data dir for the given OONI Home
func AssetsDir(home string) string {
	return filepath.Join(home, "assets")
}

// TunnelDir returns the directory where to store tunnels state
func TunnelDir(home string) string {
	return filepath.Join(home, "tunnel")
}

// EngineDir returns the directory where ooni/probe-engine should
// store its private data given a specific OONI Home.
func EngineDir(home string) string {
	return filepath.Join(home, "engine")
}

// DBDir returns the database dir for the given name
func DBDir(home string, name string) string {
	return filepath.Join(home, "db", fmt.Sprintf("%s.sqlite3", name))
}

// GetOONIHome returns the path to the OONI Home
func GetOONIHome() (string, error) {
	if ooniHome := os.Getenv("OONI_HOME"); ooniHome != "" {
		return ooniHome, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".ooniprobe")
	return path, nil
}

// DidLegacyInformedConsent checks if the user did the informed consent procedure in probe-legacy
func DidLegacyInformedConsent() bool {
	home, err := homedir.Dir()
	if err != nil {
		return false
	}

	path := filepath.Join(filepath.Join(home, ".ooni"), "initialized")
	return fsx.RegularFileExists(path)
}
