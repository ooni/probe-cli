package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RequiredDirs returns the required ooni home directories
func RequiredDirs(home string) []string {
	requiredDirs := []string{}
	requiredSubdirs := []string{"db", "msmts", "geoip"}
	for _, d := range requiredSubdirs {
		requiredDirs = append(requiredDirs, filepath.Join(home, d))
	}
	return requiredDirs
}

// GeoIPDir returns the geoip data dir for the given OONI Home
func GeoIPDir(home string) string {
	return filepath.Join(home, "geoip")
}

// DBDir returns the database dir for the given name
func DBDir(home string, name string) string {
	return filepath.Join(home, "db", fmt.Sprintf("%s.sqlite3", name))
}

// MakeResultsDir creates and returns a directory for the result
func MakeResultsDir(home string, name string, ts time.Time) (string, error) {
	p := filepath.Join(home, "msmts",
		fmt.Sprintf("%s-%s", name, ts.Format(time.RFC3339Nano)))

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
