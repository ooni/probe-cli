package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	// destdir is the -destdir flag
	destdir = flag.String("destdir", ".", "destination directory to use")

	// measurement is the -measurement flag
	measurement = flag.String("measurement", "", "measurement file to analyze")

	// mustWriteFileLn allows overwriting must.WriteFile in tests
	mustWriteFileFn = must.WriteFile

	// prefix is the -prefix flag
	prefix = flag.String("prefix", "", "prefix to add to generated files")

	// osExit allows overwriting os.Exit in tests
	osExit = os.Exit
)

func main() {
	flag.Parse()
	if *measurement == "" {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "usage: %s -measurement <file> [-prefix <prefix>]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Mini measurement processing pipeline to reprocess recent probe measurements\n")
		fmt.Fprintf(os.Stderr, "and align results calculation with ooni/data.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Analyzes the <file> provided using -measurement <file> and writes the\n")
		fmt.Fprintf(os.Stderr, "observations.json and analysis.json files in the -destdir <destdir> directory,\n")
		fmt.Fprintf(os.Stderr, "which must already exist.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Use -prefix <prefix> to add <prefix> in front of the generated files names.\n")
		fmt.Fprintf(os.Stderr, "\n")
		osExit(1)
	}

	// parse the measurement file
	var parsed minipipeline.WebMeasurement
	must.UnmarshalJSON(must.ReadFile(*measurement), &parsed)

	// generate and write observations
	observationsPath := filepath.Join(*destdir, *prefix+"observations.json")
	container := runtimex.Try1(minipipeline.IngestWebMeasurement(&parsed))
	mustWriteFileFn(observationsPath, must.MarshalAndIndentJSON(container, "", "  "), 0600)

	// generate and write observations analysis
	analysisPath := filepath.Join(*destdir, *prefix+"analysis.json")
	analysis := minipipeline.AnalyzeWebObservations(container)
	mustWriteFileFn(analysisPath, must.MarshalAndIndentJSON(analysis, "", "  "), 0600)
}
