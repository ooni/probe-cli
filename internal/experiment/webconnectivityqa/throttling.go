package webconnectivityqa

import (
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// throttlingWithHTTPS is the case where the website has throttling configured for it.
func throttlingWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "throttlingWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://largefile.com/",
		Configure: func(env *netemx.QAEnv) {

			env.DPIEngine().AddRule(&netem.DPIThrottleTrafficForTLSSNI{
				Delay:  300 * time.Millisecond,
				Logger: log.Log,
				PLR:    0.2,
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
