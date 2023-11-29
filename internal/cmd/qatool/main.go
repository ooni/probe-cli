package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivityqa"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	// destdirFlag is the -destdir flag
	destdirFlag = flag.String("destdir", "", "root directory in which to dump files")

	// disableMeasureFlag is the -disable-measure flag
	disableMeasureFlag = flag.Bool("disable-measure", false, "whether to measure again with netemx")

	// disableReprocessFlag is the -disable-reprocess flag
	disableReprocessFlag = flag.Bool("disable-reprocess", false, "whether to reprocess existing measurements")

	// helpFlag is the -help flag
	helpFlag = flag.Bool("help", false, "print help message")

	// listFlag is the -list flag
	listFlag = flag.Bool("list", false, "lists available tests")

	// mustReadFileFn allows to overwrite must.ReadFile in tests
	mustReadFileFn = must.ReadFile

	// mustWriteFileFn allows to overwrite must.WriteFile in tests
	mustWriteFileFn = must.WriteFile

	// osExitFn allows to overwrite os.Exit in tests
	osExitFn = os.Exit

	// osMkdirAllFn allows to overwrite os.MkdirAll in tests
	osMkdirAllFn = os.MkdirAll

	// runFlag is the -run flag
	runFlag = flag.String("run", "", "regexp to select which test cases to run")
)

func mustSerializeMkdirAllAndWriteFile(dirname string, filename string, content any) {
	rawData := must.MarshalAndIndentJSON(content, "", "  ")
	runtimex.Try0(osMkdirAllFn(dirname, 0700))
	mustWriteFileFn(filepath.Join(dirname, filename), rawData, 0600)
}

func runWebConnectivityLTE(tc *webconnectivityqa.TestCase) {
	// compute the actual destdir
	actualDestdir := filepath.Join(*destdirFlag, tc.Name)

	if !*disableMeasureFlag {
		// construct the proper measurer
		measurer := webconnectivitylte.NewExperimentMeasurer(&webconnectivitylte.Config{})

		// run the test case
		measurement := runtimex.Try1(webconnectivityqa.MeasureTestCase(measurer, tc))

		// serialize the original measurement
		mustSerializeMkdirAllAndWriteFile(actualDestdir, "measurement.json", measurement)
	}

	if !*disableReprocessFlag {
		// obtain the web measurement
		rawData := mustReadFileFn(filepath.Join(actualDestdir, "measurement.json"))
		var webMeasurement minipipeline.WebMeasurement
		must.UnmarshalJSON(rawData, &webMeasurement)

		// ingest web measurement
		observationsContainer := runtimex.Try1(minipipeline.IngestWebMeasurement(&webMeasurement))

		// serialize the observations
		mustSerializeMkdirAllAndWriteFile(actualDestdir, "observations.json", observationsContainer)

		// convert to classic observations
		observationsContainerClassic := minipipeline.ClassicFilter(observationsContainer)

		// serialize the classic observations
		mustSerializeMkdirAllAndWriteFile(actualDestdir, "observations_classic.json", observationsContainerClassic)

		// analyze the observations
		analysis := minipipeline.AnalyzeWebObservations(observationsContainer)

		// serialize the observations analysis
		mustSerializeMkdirAllAndWriteFile(actualDestdir, "analysis.json", analysis)

		// perform the classic analysis
		analysisClassic := minipipeline.AnalyzeWebObservations(observationsContainerClassic)

		// serialize the classic analysis results
		mustSerializeMkdirAllAndWriteFile(actualDestdir, "analysis_classic.json", analysisClassic)
	}
}

func main() {
	// parse command line flags
	flag.Parse()

	// print usage
	if *helpFlag || (*destdirFlag == "" && !*listFlag) {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "usage: %s -destdir <destdir> [-run <regexp>] [-disable-measure|-disable-reprocess]]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "       %s -list [-run <regexp>]\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "The first form of the command runs the QA tests selected by the given\n")
		fmt.Fprintf(os.Stderr, "<regexp> and creates the corresponding files in <destdir>.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "The second form of the command lists the QA tests that would be run\n")
		fmt.Fprintf(os.Stderr, "when using the given <regexp> selector.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "An empty <regepx> selector selects all QA tests.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Add the -disable-measure flag to the first form of the command to\n")
		fmt.Fprintf(os.Stderr, "avoid performing the measurements using netemx. This assums that\n")
		fmt.Fprintf(os.Stderr, "you already generated the measurements previously.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Add the -disable-reprocess flag to the first form of the command to\n")
		fmt.Fprintf(os.Stderr, "avoid reprocessing the measurements using the minipipeline.\n")
		fmt.Fprintf(os.Stderr, "\n")
		osExitFn(1)
	}

	// build the regexp
	selector := regexp.MustCompile(*runFlag)

	// select which test cases to run
	for _, tc := range webconnectivityqa.AllTestCases() {
		name := "webconnectivitylte/" + tc.Name
		if *runFlag != "" && !selector.MatchString(name) {
			continue
		}
		if *listFlag {
			fmt.Printf("%s\n", name)
			continue
		}
		runWebConnectivityLTE(tc)
	}
}
