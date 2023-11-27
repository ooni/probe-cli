package webconnectivityqa

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// httpBlockingConnectionReset verifies the case where there is a connection reset
// when the host header is emitted on the wire in cleartext.
func httpBlockingConnectionReset() *TestCase {
	return &TestCase{
		Name:  "httpBlockingConnectionReset",
		Flags: 0,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{
				Logger:          env.Logger(),
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XBlockingFlags:        8,    // analysisFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}
