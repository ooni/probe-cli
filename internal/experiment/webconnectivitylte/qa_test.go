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
				// We override the resolver used by default because the QA environment uses
				// only IP addresses in the 130.192.91.x namespace for extra robustness in case
				// netem is not working as intended and we're using the real network.
				DNSOverUDPResolver: net.JoinHostPort(netemx.QAEnvDefaultUncensoredResolverAddress, "53"),
			})
			if err := webconnectivityqa.RunTestCase(measurer, tc); err != nil {
				t.Fatal(err)
			}
		})
	}
}
