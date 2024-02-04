package webconnectivityqa

import (
	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// redirectWithConsistentDNSAndThenConnectionRefusedForHTTP is a scenario where the redirect
// works but then there's connection refused for an HTTP URL.
func redirectWithConsistentDNSAndThenConnectionRefusedForHTTP() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionRefusedForHTTP",
		Flags: TestCaseFlagNoV04,
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
			XBlockingFlags:        2, // AnalysisBlockingFlagTCPIPBlocking
			Accessible:            false,
			Blocking:              "tcp_ip",
		},
	}
}

// redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS is a scenario where the redirect
// works but then there's connection refused for an HTTPS URL.
func redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS",
		Flags: TestCaseFlagNoV04,
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
			XBlockingFlags:        2, // AnalysisBlockingFlagTCPIPBlocking
			Accessible:            false,
			Blocking:              "tcp_ip",
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
			XBlockingFlags:        12, // AnalysisBlockingFlagTLSBlocking | AnalysisBlockingFlagHTTPBlocking
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
		Flags: TestCaseFlagNoV04,
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
			XBlockingFlags:        4, // AnalysisBlockingFlagTLSBlocking
			Accessible:            false,
			Blocking:              "tls",
		},
	}
}

// redirectWithConsistentDNSAndThenNXDOMAIN is a scenario where the redirect
// works but then there's NXDOMAIN for the URL's domain
func redirectWithConsistentDNSAndThenNXDOMAIN() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenNXDOMAIN",
		Flags: 0,
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
			XBlockingFlags:        1, // AnalysisBlockingFlagDNSBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}

// redirectWithConsistentDNSAndThenEOFForHTTP is a scenario where the redirect
// works but then there's connection EOF for an HTTP URL.
func redirectWithConsistentDNSAndThenEOFForHTTP() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenEOFForHTTP",
		Flags: 0,
		Input: "https://bit.ly/32447",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot HTTP round trip
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot TLS handshake
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "eof_error",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XDNSFlags:             0,
			XBlockingFlags:        12, // AnalysisBlockingFlagTLSBlocking | AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenEOFForHTTPS is a scenario where the redirect
// works but then there's connection EOF for an HTTPS URL.
func redirectWithConsistentDNSAndThenEOFForHTTPS() *TestCase {
	return &TestCase{
		Name:  "redirectWithConsistentDNSAndThenEOFForHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://bit.ly/21645",
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot connect to the example domain on port 80
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot connect to the example domain on port 443
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "eof_error",
			XStatus:               8448, // StatusExperimentHTTP | StatusAnomalyReadWrite
			XDNSFlags:             0,
			XBlockingFlags:        4, // AnalysisBlockingFlagTLSBlocking
			Accessible:            false,
			Blocking:              "tls",
		},
	}
}

// redirectWithConsistentDNSAndThenTimeoutForHTTP is a scenario where the redirect
// works but then there's connection refused for an HTTP URL.
func redirectWithConsistentDNSAndThenTimeoutForHTTP() *TestCase {
	return &TestCase{
		Name:     "redirectWithConsistentDNSAndThenTimeoutForHTTP",
		Flags:    0,
		Input:    "https://bit.ly/32447",
		LongTest: true,
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot perform the round trip
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot TLS handshake
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "generic_timeout_error",
			XStatus:               8704, // StatusExperimentHTTP | StatusAnomalyUnknown
			XDNSFlags:             0,
			XBlockingFlags:        12, // AnalysisBlockingFlagTLSBlocking | AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithConsistentDNSAndThenTimeoutForHTTPS is a scenario where the redirect
// works but then there's connection refused for an HTTPS URL.
func redirectWithConsistentDNSAndThenTimeoutForHTTPS() *TestCase {
	return &TestCase{
		Name:     "redirectWithConsistentDNSAndThenTimeoutForHTTPS",
		Flags:    TestCaseFlagNoV04,
		Input:    "https://bit.ly/21645",
		LongTest: true,
		Configure: func(env *netemx.QAEnv) {

			// make sure we cannot HTTP round trip
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForString{
				Logger:          log.Log,
				ServerIPAddress: netemx.AddressWwwExampleCom,
				ServerPort:      80,
				String:          "www.example.com",
			})

			// make sure we cannot TLS handshake
			env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "generic_timeout_error",
			XStatus:               8704, // StatusExperimentHTTP | StatusAnomalyUnknown
			XDNSFlags:             0,
			XBlockingFlags:        4, // AnalysisBlockingFlagTLSBlocking
			Accessible:            false,
			Blocking:              "tls",
		},
	}
}

// redirectWithBrokenLocationForHTTP is a scenario where the redirect
// returns a broken URL only containing `http://`.
//
// See https://github.com/ooni/probe/issues/2628 for more info.
func redirectWithBrokenLocationForHTTP() *TestCase {
	return &TestCase{
		Name:     "redirectWithBrokenLocationForHTTP",
		Flags:    TestCaseFlagNoV04,
		Input:    "http://httpbin.com/broken-redirect-http",
		LongTest: true,
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: netxlite.FailureHTTPInvalidRedirectLocationHost,
			XStatus:               8192, // StatusExperimentHTTP
			XDNSFlags:             0,
			XBlockingFlags:        8, // AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}

// redirectWithBrokenLocationForHTTPS is a scenario where the redirect
// returns a broken URL only containing `https://`.
//
// See https://github.com/ooni/probe/issues/2628 for more info.
func redirectWithBrokenLocationForHTTPS() *TestCase {
	return &TestCase{
		Name:     "redirectWithBrokenLocationForHTTPS",
		Flags:    TestCaseFlagNoV04,
		Input:    "https://httpbin.com/broken-redirect-https",
		LongTest: true,
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: netxlite.FailureHTTPInvalidRedirectLocationHost,
			XStatus:               8192, // StatusExperimentHTTP
			XDNSFlags:             0,
			XBlockingFlags:        8, // AnalysisBlockingFlagHTTPBlocking
			Accessible:            false,
			Blocking:              "http-failure",
		},
	}
}
