package testingx_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/testingproxy"
)

func TestHTTPProxyHandler(t *testing.T) {
	for _, testCase := range testingproxy.AllTestCases {
		short := testCase.Short()
		if !short && testing.Short() {
			t.Skip("skip test in short mode")
		}
		t.Run(testCase.Name(), func(t *testing.T) {
			testCase.Run(t)
		})
	}
}
