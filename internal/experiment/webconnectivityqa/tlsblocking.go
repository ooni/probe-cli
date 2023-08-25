package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// tlsBlockingConnectionReset resets the connection for the SNI we're using.
func tlsBlockingConnectionReset() *TestCase {
	return &TestCase{
		Name:  "tlsBlockingConnectionReset",
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {
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
			XBlockingFlags:        4,    // analysisFlagTLSBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}
