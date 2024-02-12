package webconnectivitylte

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/webconnectivityqa"
)

func TestQA(t *testing.T) {
	SortObservations.Add(1) // make sure we have predictable observations
	for _, tc := range webconnectivityqa.AllTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			if (tc.Flags & webconnectivityqa.TestCaseFlagNoLTE) != 0 {
				t.Skip("this test case cannot run on Web Connectivity LTE")
			}
			if testing.Short() && tc.LongTest {
				t.Skip("skip test in short mode")
			}
			measurer := NewExperimentMeasurer(&Config{})
			if err := webconnectivityqa.RunTestCase(measurer, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
