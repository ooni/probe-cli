package minipipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/optional"
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
				gotContainerData, err := minipipeline.IngestWebMeasurement(
					model.GeoIPASNLookupperFunc(geoipx.LookupASN),
					&measurementData,
				)
				if err != nil {
					t.Fatal(err)
				}

				// convert the container into a classic container
				gotClassicContainerData := minipipeline.ClassicFilter(gotContainerData)

				// analyze the measurement
				gotAnalysisData := minipipeline.AnalyzeWebObservationsWithLinearAnalysis(
					model.GeoIPASNLookupperFunc(geoipx.LookupASN),
					gotContainerData,
				)

				// perform the classic web-connectivity-v0.4-like analysis
				gotClassicAnalysisData := minipipeline.AnalyzeWebObservationsWithLinearAnalysis(
					model.GeoIPASNLookupperFunc(geoipx.LookupASN),
					gotClassicContainerData,
				)

				//
				// Note: if tests fail, you likely need to regenerate the static test
				// cases using ./script/updateminipipeline.bash and you should also eyeball
				// the diff for these changes to see if it makes sense.
				//

				t.Run("linear consistency checks", func(t *testing.T) {
					testConsistencyChecksForLinear(t, gotAnalysisData.Linear)
					testConsistencyChecksForLinear(t, gotClassicAnalysisData.Linear)
				})

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

func testConsistencyChecksForLinear(t *testing.T, linear []*minipipeline.WebObservation) {
	// Here are the checks:
	//
	// 1. the TagDepth MUST decrease monotonically
	//
	// 2. the Type MUST decrease monotonically within the TagDepth
	//
	// 3. errors MUST appear after successes within the TagDepth

	var (
		currentTagDepth      optional.Value[int64]
		currentType          optional.Value[int64]
		currentlyInsideError bool
	)

	for _, entry := range linear {
		t.Log("currently processing", string(must.MarshalJSON(entry)))

		// make sure the messages are reasonably well formed
		if entry.TagDepth.IsNone() {
			t.Fatal("expected to have TagDepth for all entries in the test suite, found", string(must.MarshalJSON(entry)))
		}
		if entry.Failure.IsNone() {
			t.Fatal("expected to have Failure for all entries in the test suite, found", string(must.MarshalJSON(entry)))
		}

		// initialize if needed
		runtimex.Assert(
			(currentTagDepth.IsNone() && currentType.IsNone()) ||
				(!currentTagDepth.IsNone() && !currentType.IsNone()),
			"expected currentTagDepth and currentType to be in sync here",
		)
		if currentTagDepth.IsNone() {
			currentTagDepth = entry.TagDepth
			currentType = optional.Some(int64(entry.Type))
			currentlyInsideError = false
		}

		// make sure there's monotonic decrease of the current tag depth
		// and adjust the state in case there is an actual decrease
		if entry.TagDepth.Unwrap() > currentTagDepth.Unwrap() {
			t.Fatal("there should not be an increase in the TagDepth", string(must.MarshalJSON(entry)))
		}
		if entry.TagDepth.Unwrap() < currentTagDepth.Unwrap() {
			currentTagDepth = entry.TagDepth
			currentType = optional.Some(int64(entry.Type))
			currentlyInsideError = false
		}

		// make sure there's monotonic decrease of the current type
		if int64(entry.Type) > currentType.Unwrap() {
			t.Fatal("there should not be an increase of a Type within a TagDepth", string(must.MarshalJSON(entry)))
		}
		if int64(entry.Type) < currentType.Unwrap() {
			currentType = optional.Some(int64(entry.Type))
			currentlyInsideError = false
		}

		// make sure we cannot go back to success if we're inside error
		if currentlyInsideError && entry.Failure.Unwrap() == "" {
			t.Fatal("we cannot go from error to not error within a given Type", string(must.MarshalJSON(entry)))
		}
		if !currentlyInsideError && entry.Failure.Unwrap() != "" {
			currentlyInsideError = true
		}
	}
}

func TestQAWeb(t *testing.T) {
	testMustRunAllWebTestCases(t, filepath.Join("testdata", "webconnectivity", "generated"))
	testMustRunAllWebTestCases(t, filepath.Join("testdata", "webconnectivity", "manual"))
}
