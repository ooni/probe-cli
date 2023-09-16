package testingproxy_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/testingproxy"
)

func TestHTTPProxy(t *testing.T) {
	for _, testCase := range testingproxy.HTTPTestCases {
		t.Run(testCase.Name(), func(t *testing.T) {
			short := testCase.Short()
			if !short && testing.Short() {
				t.Skip("skip test in short mode")
			}
			testCase.Run(t)
		})
	}
}

func TestSOCKSProxy(t *testing.T) {
	for _, testCase := range testingproxy.SOCKSTestCases {
		t.Run(testCase.Name(), func(t *testing.T) {
			short := testCase.Short()
			if !short && testing.Short() {
				t.Skip("skip test in short mode")
			}
			testCase.Run(t)
		})
	}
}
