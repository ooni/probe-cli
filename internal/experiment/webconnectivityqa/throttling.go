package webconnectivityqa

import (
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// throttlingWithHTTP is the case where an HTTP website has throttling configured for it.
func throttlingWithHTTP() *TestCase {
	return &TestCase{
		Name:  "throttlingWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://largefile.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPIThrottleTrafficForTCPEndpoint{
				Delay:           300 * time.Millisecond,
				Logger:          log.Log,
				PLR:             0.1,
				ServerIPAddress: netemx.AddressLargeFileCom1,
				ServerPort:      80,
			})

			env.DPIEngine().AddRule(&netem.DPIThrottleTrafficForTCPEndpoint{
				Delay:           300 * time.Millisecond,
				Logger:          log.Log,
				PLR:             0.1,
				ServerIPAddress: netemx.AddressLargeFileCom2,
				ServerPort:      80,
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "generic_timeout_error",
			XBlockingFlags:        8, // AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// throttlingWithHTTPS is the case where an HTTPS website has throttling configured for it.
func throttlingWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "throttlingWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://largefile.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPIThrottleTrafficForTLSSNI{
				Delay:  300 * time.Millisecond,
				Logger: log.Log,
				PLR:    0.1,
				SNI:    "largefile.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "generic_timeout_error",
			XBlockingFlags:        8, // AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}
