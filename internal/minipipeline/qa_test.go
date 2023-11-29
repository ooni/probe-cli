package minipipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func testCmpDiffUsingGenericMaps(origLeft, origRight any) string {
	rawLeft := must.MarshalJSON(origLeft)
	rawRight := must.MarshalJSON(origRight)
	var left map[string]any
	must.UnmarshalJSON(rawLeft, &left)
	var right map[string]any
	must.UnmarshalJSON(rawRight, &right)
	return cmp.Diff(left, right)
}

func testMustRunAllWebTestCases(t *testing.T, topdir string) {
	t.Run(topdir, func(t *testing.T) {
		entries := runtimex.Try1(os.ReadDir(topdir))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			t.Run(entry.Name(), func(t *testing.T) {
				fullpath := filepath.Join(topdir, entry.Name())
				// read the raw measurement from the test case
				measurementFile := filepath.Join(fullpath, "measurement.json")
				measurementRaw := must.ReadFile(measurementFile)
				var measurementData minipipeline.WebMeasurement
				must.UnmarshalJSON(measurementRaw, &measurementData)

				// load the expected container from the test case
				expectedContainerFile := filepath.Join(fullpath, "observations.json")
				expectedContainerRaw := must.ReadFile(expectedContainerFile)
				var expectedContainerData minipipeline.WebObservationsContainer
				must.UnmarshalJSON(expectedContainerRaw, &expectedContainerData)

				// load the expected classic container from the test case
				expectedClassicContainerFile := filepath.Join(fullpath, "observations_classic.json")
				expectedClassicContainerRaw := must.ReadFile(expectedClassicContainerFile)
				var expectedClassicContainerData minipipeline.WebObservationsContainer
				must.UnmarshalJSON(expectedClassicContainerRaw, &expectedClassicContainerData)

				// load the expected analysis from the test case
				expectedAnalysisFile := filepath.Join(fullpath, "analysis.json")
				expectedAnalysisRaw := must.ReadFile(expectedAnalysisFile)
				var expectedAnalysisData minipipeline.WebAnalysis
				must.UnmarshalJSON(expectedAnalysisRaw, &expectedAnalysisData)

				// load the expected classic analysis from the test case
				expectedClassicAnalysisFile := filepath.Join(fullpath, "analysis_classic.json")
				expectedClassicAnalysisRaw := must.ReadFile(expectedClassicAnalysisFile)
				var expectedClassicAnalysisData minipipeline.WebAnalysis
				must.UnmarshalJSON(expectedClassicAnalysisRaw, &expectedClassicAnalysisData)

				// load the measurement into the pipeline
				gotContainerData, err := minipipeline.IngestWebMeasurement(&measurementData)
				if err != nil {
					t.Fatal(err)
				}

				// convert the container into a classic container
				gotClassicContainerData := minipipeline.ClassicFilter(gotContainerData)

				// analyze the measurement
				gotAnalysisData := minipipeline.AnalyzeWebObservations(gotContainerData)

				// perform the classic web-connectivity-v0.4-like analysis
				gotClassicAnalysisData := minipipeline.AnalyzeWebObservations(gotClassicContainerData)

				t.Run("observations", func(t *testing.T) {
					if diff := testCmpDiffUsingGenericMaps(&expectedContainerData, gotContainerData); diff != "" {
						t.Fatal(diff)
					}
				})

				t.Run("observations_classic", func(t *testing.T) {
					if diff := testCmpDiffUsingGenericMaps(&expectedClassicContainerData, gotClassicContainerData); diff != "" {
						t.Fatal(diff)
					}
				})

				t.Run("analysis", func(t *testing.T) {
					if diff := testCmpDiffUsingGenericMaps(&expectedAnalysisData, gotAnalysisData); diff != "" {
						t.Fatal(diff)
					}
				})

				t.Run("analysis_classic", func(t *testing.T) {
					if diff := testCmpDiffUsingGenericMaps(&expectedClassicAnalysisData, gotClassicAnalysisData); diff != "" {
						t.Fatal(diff)
					}
				})
			})
		}
	})
}

func TestQAWeb(t *testing.T) {
	testMustRunAllWebTestCases(t, filepath.Join("testdata", "webconnectivity", "generated"))
	testMustRunAllWebTestCases(t, filepath.Join("testdata", "webconnectivity", "manual"))
}
