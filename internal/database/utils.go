package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// ResultTimestamp is a windows friendly timestamp
const ResultTimestamp = "2006-01-02T150405.999999999Z0700"

// MakeResultsDir creates and returns a directory for the result
func makeResultsDir(home string, name string, ts time.Time) (string, error) {
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
