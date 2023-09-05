package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// redirectWithConsistentDNSAndThenConnectionRefusedForHTTP is a scenario where the redirect
// works but then there's connection refused for an HTTP URL.
func redirectWithConsistentDNSAndThenConnectionRefusedForHTTP() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionRefusedForHTTP",
		Flags: TestCaseFlagNoLTE, // BUG: LTE thinks this website is accessible (WTF?!)
		Input: "https://bit.ly/32447",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot connect to the example domain on port 80
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
			})

			// make sure we cannot connect to the example domain on port 443
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      443,
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_refused",
			XStatus:               8320, // StatusExperimentHTTP | StatusAnomalyConnect
			XDNSFlags:             0,
			XBlockingFlags:        32, // analysisFlagSuccess
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS is a scenario where the redirect
// works but then there's connection refused for an HTTPS URL.
func redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS",
		Flags: TestCaseFlagNoLTE, // BUG: LTE thinks this website is accessible (WTF?!)
		Input: "https://bit.ly/21645",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot connect to the example domain on port 80
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
			})

			// make sure we cannot connect to the example domain on port 443
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      443,
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_refused",
			XStatus:               8320, // StatusExperimentHTTP | StatusAnomalyConnect
			XDNSFlags:             0,
			XBlockingFlags:        32, // analysisFlagSuccess
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenConnectionResetForHTTP is a scenario where the redirect
// works but then there's connection refused for an HTTP URL.
func redirectWithConsistentDNSAndThenConnectionResetForHTTP() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionResetForHTTP",
		Flags: 0,
		Input: "https://bit.ly/32447",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot HTTP round trip
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot TLS handshake
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XDNSFlags:             0,
			XBlockingFlags:        8, // analysisFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenConnectionResetForHTTPS is a scenario where the redirect
// works but then there's connection refused for an HTTPS URL.
func redirectWithConsistentDNSAndThenConnectionResetForHTTPS() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionResetForHTTPS",
		Flags: TestCaseFlagNoLTE, // BUG: LTE thinks this website is accessible (WTF?!)
		Input: "https://bit.ly/21645",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot HTTP round trip
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot TLS handshake
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "connection_reset",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XDNSFlags:             0,
			XBlockingFlags:        8, // analysisFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenNXDOMAIN is a scenario where the redirect
// works but then there's NXDOMAIN for the URL's domain
func redirectWithConsistentDNSAndThenNXDOMAIN() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenNXDOMAIN",
		Flags: TestCaseFlagNoLTE, // BUG: LTE thinks this website is accessible (WTF?!)
		Input: "https://bit.ly/21645",
		Configure: func(env *netemx.QAEnv) {

			// Empty addresses cause NXDOMAIN
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{},
				Logger:    env.Logger(),
				Domain:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "dns_nxdomain_error",
			XStatus:               8224, // StatusExperimentHTTP | StatusAnomalyDNS
			XDNSFlags:             0,
			XBlockingFlags:        8, // analysisFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
