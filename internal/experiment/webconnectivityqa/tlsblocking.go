package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// tlsBlockingConnectionResetWithConsistentDNS resets the connection for the SNI we're using.
func tlsBlockingConnectionResetWithConsistentDNS() *TestCase {
	return &TestCase{
		Name:  "tlsBlockingConnectionResetWithConsistentDNS",
		Flags: TestCaseFlagNoV04,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{
				Logger: log.Log,
				String: "www.example.com",
			})

			env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XBlockingFlags:        4,    // AnalysisBlockingFlagTLSBlocking
			Accessible:            false,
			Blocking:              "tls",
		},
	}
}

// tlsBlockingConnectionResetWithInconsistentDNS resets the connection for the SNI we're using.
func tlsBlockingConnectionResetWithInconsistentDNS() *TestCase {
	return &TestCase{
		Name:  "tlsBlockingConnectionResetWithInconsistentDNS",
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{
					netemx.ISPProxyAddress,
				},
				Logger: log.Log,
				Domain: "www.example.com",
			})

			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{
				Logger: log.Log,
				String: "www.example.com",
			})

			env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "inconsistent",
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8480, // StatusExperimentHTTP | StatusAnomalyReadWrite | StatusAnomalyDNS
			XDNSFlags:             4,    // AnalysisDNSFlagUnexpectedAddrs
			XBlockingFlags:        5,    // AnalysisBlockingFlagTLSBlocking | AnalysisBlockingFlagDNSBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
