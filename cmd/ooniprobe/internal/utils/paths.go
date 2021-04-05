package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils/homedir"
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

// FileExists returns true if the specified path exists and is a
// regular file.
func FileExists(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && stat.Mode().IsRegular()
}

// ResultTimestamp is a windows friendly timestamp
const ResultTimestamp = "2006-01-02T150405.999999999Z0700"

// MakeResultsDir creates and returns a directory for the result
func MakeResultsDir(home string, name string, ts time.Time) (string, error) {
	p := filepath.Join(home, "msmts",
		fmt.Sprintf("%s-%s", name, ts.Format(ResultTimestamp)))

	// If the path already exists, this is a problem. It should not clash, because
	// we are using nanosecond precision for the starttime.
	if _, e := os.Stat(p); e == nil {
		return "", errors.New("results path already exists")
	}
	err := os.MkdirAll(p, 0700)
	if err != nil {
		return "", err
	}
	return p, nil
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
	if FileExists(path) {
		return true
	}
	return false
}
