package webconnectivity

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivityqa"
)

func TestQA(t *testing.T) {
	for _, tc := range webconnectivityqa.AllTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			measurer := NewExperimentMeasurer(Config{})
			if err := webconnectivityqa.RunTestCase(measurer, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
