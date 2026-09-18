package webconnectivity

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/webconnectivityqa"
)

func TestQA(t *testing.T) {
	for _, tc := range webconnectivityqa.AllTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			if (tc.Flags & webconnectivityqa.TestCaseFlagNoV04) != 0 {
				t.Skip("this test case cannot run on Web Connectivity v0.4")
			}
			if testing.Short() && tc.LongTest {
				t.Skip("skip test in short mode")
			}
			measurer := NewExperimentMeasurer()
			newTarget := func(input string) model.ExperimentTarget {
				return &Target{URL: input, Config: &Config{}}
			}
			if err := webconnectivityqa.RunTestCase(measurer, newTarget, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
