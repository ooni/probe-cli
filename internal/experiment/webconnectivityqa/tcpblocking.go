package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// tcpBlockingConnectTimeout verifies that we correctly handle the case
// where the connection is timed out.
func tcpBlockingConnectTimeout() *TestCase {
	return &TestCase{
		Name:  "tcpBlockingConnectTimeout",
		Flags: 0,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: netemx.InternetScenarioAddressWwwExampleCom,
				ServerPort:      443,
				ServerProtocol:  layers.IPProtocolTCP,
			})
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "generic_timeout_error",
			XStatus:               4224, // StatusAnomalyConnect | StatusExperimentConnect
			XBlockingFlags:        2,    // analysisFlagTCPIPBlocking
			Accessible:            false,
			Blocking:              "tcp_ip",
		},
	}
}
