package testingproxy_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/testingproxy"
)

func TestWorkingAsIntended(t *testing.T) {
	for _, testCase := range testingproxy.AllTestCases {
		t.Run(testCase.Name(), func(t *testing.T) {
			short := testCase.Short()
			if !short && testing.Short() {
				t.Skip("skip test in short mode")
			}
			testCase.Run(t)
		})
	}
}
