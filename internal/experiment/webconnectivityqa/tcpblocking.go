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

// tcpBlockingConnectionRefusedWithInconsistentDNS verifies that we correctly
// handle the case where the DNS is inconsistent and there's TCP blocking.
func tcpBlockingConnectionRefusedWithInconsistentDNS() *TestCase {
	return &TestCase{
		Name:  "tcpBlockingConnectionRefusedWithInconsistentDNS",
		Flags: 0,
		Input: "https://www.example.org/",
		Configure: func(env *netemx.QAEnv) {

			// spoof the DNS response to force using example.com's address
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{"130.192.91.7"},
				Logger:    log.Log,
				Domain:    "www.example.org",
			})

			// make sure we cannot connect to example.com
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: "130.192.91.7", // www.example.com
				ServerPort:      443,
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "inconsistent",
			HTTPExperimentFailure: "connection_refused",
			XStatus:               4256, // StatusExperimentConnect | StatusAnomalyConnect | StatusAnomalyDNS
			XDNSFlags:             4,    // AnalysisDNSUnexpectedAddrs
			XBlockingFlags:        35,   // analysisFlagSuccess | analysisFlagDNSBlocking | analysisFlagTCPIPBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
