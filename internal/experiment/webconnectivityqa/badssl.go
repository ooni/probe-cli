package webconnectivityqa

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// Sometimes people we measure the websites of let their certificates expire and
// we want to be confident about correctly measuring this condition
func badSSLWithExpiredCertificate() *TestCase {
	return &TestCase{
		Name:  "badSSLWithExpiredCertificate",
		Flags: 0,
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
		},
	}
}

// Sometimes people we measure the websites of misconfigured the certificate names and
// we want to be confident about correctly measuring this condition
func badSSLWithWrongServerName() *TestCase {
	return &TestCase{
		Name:  "badSSLWithWrongServerName",
		Flags: 0,
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
		},
	}
}

// Let's be sure we correctly flag a website using an unknown-to-us authority.
func badSSLWithUnknownAuthorityWithConsistentDNS() *TestCase {
	return &TestCase{
		Name:  "badSSLWithUnknownAuthorityWithConsistentDNS",
		Flags: 0,
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
		},
	}
}

// This test case models when we're redirected to a blockpage website using a custom CA.
func badSSLWithUnknownAuthorityWithInconsistentDNS() *TestCase {
	return &TestCase{
		Name:  "badSSLWithUnknownAuthorityWithInconsistentDNS",
		Flags: 0,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// add DPI rule to force all the cleartext DNS queries to
			// point the client to use the ISPProxyAddress
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{netemx.AddressBadSSLCom},
				Logger:    env.Logger(),
				Domain:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:        "inconsistent",
			HTTPExperimentFailure: "ssl_unknown_authority",
			XStatus:               9248, // StatusExperimentHTTP | StatusAnomalyTLSHandshake | StatusAnomalyDNS
			XDNSFlags:             4,    // AnalysisDNSUnexpectedAddrs
			XBlockingFlags:        1,    // analysisFlagDNSBlocking
			Accessible:            false,
			Blocking:              "dns",
		},
	}
}
