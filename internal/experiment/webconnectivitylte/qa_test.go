package webconnectivitylte

import (
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivityqa"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestQA(t *testing.T) {
	for _, tc := range webconnectivityqa.AllTestCases() {
		t.Run(tc.Name, func(t *testing.T) {
			if (tc.Flags & webconnectivityqa.TestCaseFlagNoLTE) != 0 {
				t.Skip("this nettest cannot run on Web Connectivity LTE")
			}
			measurer := NewExperimentMeasurer(&Config{
				// We override the resolver to use the one we should be using with netem
				DNSOverUDPResolver: net.JoinHostPort(netemx.DefaultUncensoredResolverAddress, "53"),
			})
			if err := webconnectivityqa.RunTestCase(measurer, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
