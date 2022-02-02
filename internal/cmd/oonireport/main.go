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
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/pborman/getopt/v2"
)

// ./submit -F /home/kelmenhorst/fellowship/measurements/IR/20220116.jsonl [--control]

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
	path    string
	control bool
)

func fatalIfFalse(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func canOpen(filepath string) bool {
	stat, err := os.Stat(filepath)
	return err == nil && stat.Mode().IsRegular()
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
	fatalIfFalse(len(args) == 2, "Usage: ./oonireport upload <file>")
	fatalIfFalse(args[0] == "upload", "Unsupported operation")
	fatalIfFalse(canOpen(args[1]), "Cannot open measurement file")

	path = args[1]

	ctx := context.Background()
	logger := &log.Logger{Level: log.InfoLevel, Handler: &logHandler{Writer: os.Stderr}}

	// create new session
	config := engine.SessionConfig{
		Logger:          logger,
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
	}
	sess, err := engine.NewSession(ctx, config)
	defer sess.Close()

	// open measurement file
	file, err := os.Open(path)
	if err != nil {
		fmt.Println("error while trying to open file", err)
		return
	}

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
	file.Close()

	submitted := 0
	for _, measurement := range lines {
		// create probe services client and submitter
		psc, err := sess.NewProbeServicesClient(ctx)
		if err != nil {
			fmt.Println("error occurred while creating client", err)
			os.Exit(0)
		}
		submitter := probeservices.NewSubmitter(psc, sess.Logger())

		// load input as model.Measurement
		var mm model.Measurement
		if err := json.Unmarshal([]byte(measurement), &mm); err != nil {
			fmt.Println("error occurred at json.Unmarshal", err)
			os.Exit(0)
		}
		// submit the measurement
		if err := submitter.Submit(ctx, &mm); err != nil {
			fmt.Println("error occurred while submitting", err)
			os.Exit(0)
		}
		submitted += 1
		json.Marshal(mm)
		runtimex.PanicOnError(err, "json.Marshal should not fail here")
		fmt.Println(mm.ReportID)
	}
	fmt.Println("Submitted measurements: ", submitted)
}
