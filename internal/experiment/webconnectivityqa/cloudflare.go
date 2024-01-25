package webconnectivityqa

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// cloudflareCAPTCHAWithHTTP obtains the cloudflare CAPTCHA using HTTP.
func cloudflareCAPTCHAWithHTTP() *TestCase {
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
			DNSConsistency: "consistent",
			// TODO(bassosimone): it seems LTE QA does not check for the value of
			// the HTTPExperimentFailure field, why?
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XBlockingFlags:        8,    // AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}
