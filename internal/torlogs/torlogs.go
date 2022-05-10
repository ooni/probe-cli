// Package torlogs contains code to read tor logs.
package torlogs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	// ErrEmptyLogFilePath indicates that the log file path is empty.
	ErrEmptyLogFilePath = errors.New("torlogs: empty log file path")

	// ErrCannotReadLogFile indicates we cannot read the log file.
	ErrCannotReadLogFile = errors.New("torlogs: cannot read the log file")

	// ErrNoBootstrapLogs indicates we could not find any bootstrap log in the log file.
	ErrNoBootstrapLogs = errors.New("torlogs: no bootstrap logs")

	// ErrCannotFindSubmatches indicates we cannot find submatches.
	ErrCannotFindSubmatches = errors.New("torlogs: cannot find submatches")
)

// torBootstrapRegexp helps to extract progress info from logs.
//
// See https://regex101.com/r/Do07qd/1.
var torBootstrapRegexp = regexp.MustCompile(
	`^[A-Za-z0-9.: ]+ \[notice\] Bootstrapped ([0-9]+)% \(([A-Za-z_]+)\): ([A-Za-z0-9 ]+)$`)

// ReadBootstrapLogs reads tor logs from the given file and
// returns a list of bootstrap-related logs.
func ReadBootstrapLogs(logFilePath string) ([]string, error) {
	// Implementation note:
	//
	// Tor is know to be good software that does not break its output
	// unnecessarily and that does not include PII into its logs unless
	// explicitly asked to. This fact gives me confidence that we can
	// safely include this subset of the logs into the results.
	//
	// On this note, I think it's safe to include timestamps from the
	// logs into the output, since we have a timestamp for the whole
	// experiment already, so we don't leak much more by also including
	// the Tor proper timestamps into the results.
	if logFilePath == "" {
		return nil, ErrEmptyLogFilePath
	}
	data, err := os.ReadFile(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotReadLogFile, err.Error())
	}
	var out []string
	for _, bline := range bytes.Split(data, []byte("\n")) {
		if torBootstrapRegexp.Match(bline) {
			out = append(out, string(bline))
		}
	}
	if len(out) <= 0 {
		return nil, ErrNoBootstrapLogs
	}
	return out, nil
}

// ReadBootstrapLogsOrWarn is like ReadBootstrapLogs except that it does
// not return an error on failure, rather it emits a warning.
func ReadBootstrapLogsOrWarn(logger model.Logger, logFilePath string) []string {
	logs, err := ReadBootstrapLogs(logFilePath)
	if err != nil {
		logger.Warnf("%s", err.Error())
		return nil
	}
	return logs
}

// BootstrapInfo contains info extracted from a bootstrap log line.
type BootstrapInfo struct {
	// Progress is the progress (between 0 and 100)
	Progress int64

	// Tag is the machine readable description of the bootstrap state.
	Tag string

	// Summary is the human readable summary.
	Summary string
}

// ParseBootstrapLogLine takes in input a bootstrap log line and returns
// in output the components of such a log line.
func ParseBootstrapLogLine(logLine string) (*BootstrapInfo, error) {
	values := torBootstrapRegexp.FindStringSubmatch(logLine)
	if len(values) != 4 {
		return nil, ErrCannotFindSubmatches
	}
	progress, _ := strconv.ParseInt(values[1], 10, 64)
	bi := &BootstrapInfo{
		Progress: progress,
		Tag:      values[2],
		Summary:  values[3],
	}
	return bi, nil
}
