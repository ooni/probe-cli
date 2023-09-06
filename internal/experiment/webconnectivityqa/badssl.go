package webconnectivityqa

import "github.com/ooni/probe-cli/v3/internal/netemx"

// Sometimes people we measure the websites of let their certificates expire and
// we want to be confident about correctly measuring this condition
func badSSLWithExpiredCertificate() *TestCase {
	return &TestCase{
		Name:  "badSSLWithExpiredCertificate",
		Flags: TestCaseFlagNoLTE, // LTE flags it correctly but let's focus on v0.4 for now
		Input: "https://expired.badssl.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "ssl_invalid_certificate",
			XStatus:               16, // StatusAnomalyControlFailure
			XNullNullFlags:        4,  // analysisFlagNullNullTLSMisconfigured
			Accessible:            nil,
			Blocking:              nil,
		},
	}
}

// Sometimes people we measure the websites of misconfigured the certificate names and
// we want to be confident about correctly measuring this condition
func badSSLWithWrongServerName() *TestCase {
	return &TestCase{
		Name:  "badSSLWithWrongServerName",
		Flags: TestCaseFlagNoLTE, // LTE flags it correctly but let's focus on v0.4 for now
		Input: "https://wrong.host.badssl.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "ssl_invalid_hostname",
			XStatus:               16, // StatusAnomalyControlFailure
			XNullNullFlags:        4,  // analysisFlagNullNullTLSMisconfigured
			Accessible:            nil,
			Blocking:              nil,
		},
	}
}

// Let's be sure we correctly flag a website using an unknown-to-us authority.
func badSSLWithUnknownAuthority() *TestCase {
	return &TestCase{
		Name:  "badSSLWithUnknownAuthority",
		Flags: TestCaseFlagNoLTE, // LTE flags it correctly but let's focus on v0.4 for now
		Input: "https://untrusted-root.badssl.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: "ssl_unknown_authority",
			XStatus:               16, // StatusAnomalyControlFailure
			XNullNullFlags:        4,  // analysisFlagNullNullTLSMisconfigured
			Accessible:            nil,
			Blocking:              nil,
		},
	}
}
