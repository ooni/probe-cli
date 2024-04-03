// Command oonireport uploads reports stored on disk to the OONI collector.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/apex/log"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/pborman/getopt/v2"
)

var startTime = time.Now()

type logHandler struct {
	io.Writer
}

func (h *logHandler) HandleLog(e *log.Entry) (err error) {
	s := fmt.Sprintf("[%14.6f] <%s> %s", time.Since(startTime).Seconds(), e.Level, e.Message)
	if len(e.Fields) > 0 {
		s += fmt.Sprintf(": %+v", e.Fields)
	}
	s += "\n"
	_, err = h.Writer.Write([]byte(s))
	return
}

const (
	softwareName    = "miniooni"
	softwareVersion = version.Version
)

var (
	path string
)

func fatalIfFalse(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func readLines(path string) []string {
	// open measurement file
	file, err := os.Open(path)
	runtimex.PanicOnError(err, "Open file error.")
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// the maximum line length should be selected really big
	const maxCapacity = 800000
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	// scan measurement file, one measurement per line
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}
	return lines
}

// newSession creates a new session
func newSession(ctx context.Context) *engine.Session {
	logger := &log.Logger{Level: log.InfoLevel, Handler: &logHandler{Writer: os.Stderr}}

	config := engine.SessionConfig{
		Logger:          logger,
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
	}
	sess, err := engine.NewSession(ctx, config)
	runtimex.PanicOnError(err, "Error when trying to create session.")
	return sess
}

// new Submitter creates a probe services client and submitter
func newSubmitter(sess *engine.Session, ctx context.Context) model.Submitter {
	return runtimex.Try1(sess.NewSubmitter(ctx))
}

// toMeasurement loads an input string as model.Measurement
func toMeasurement(s string) *model.Measurement {
	var mm model.Measurement
	err := json.Unmarshal([]byte(s), &mm)
	runtimex.PanicOnError(err, "json.Unmarshal error")
	return &mm
}

// submitAll submits the measurements in input. Returns the count of submitted measurements, both
// on success and on error, and the error that occurred (nil on success).
func submitAll(ctx context.Context, lines []string, subm model.Submitter) (int, error) {
	submitted := 0
	for _, line := range lines {
		mm := toMeasurement(line)
		// submit the measurement
		err := subm.Submit(ctx, mm)
		if err != nil {
			return submitted, err
		}
		submitted += 1
	}
	return submitted, nil
}

func mainWithArgs(args []string) {
	fatalIfFalse(len(args) == 2, "Usage: ./oonireport upload <file>")
	fatalIfFalse(args[0] == "upload", "Unsupported operation")
	fatalIfFalse(fsx.RegularFileExists(args[1]), "Cannot open measurement file")

	path = args[1]
	lines := readLines(path)

	ctx := context.Background()
	sess := newSession(ctx)
	defer sess.Close()

	submitter := newSubmitter(sess, ctx)

	n, err := submitAll(ctx, lines, submitter)
	fmt.Println("Submitted measurements: ", n)
	runtimex.PanicOnError(err, "error occurred while submitting")
}

func main() {
	defer func() {
		if s := recover(); s != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %s\n", s)
		}
	}()
	// parse command line arguments
	getopt.Parse()
	args := getopt.Args()
	mainWithArgs(args)
}
