package webconnectivityqa

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// dnsHijackingToProxyWithHTTPURL is the case where an ISP rule forces clients to always
// use an explicity passthrough proxy for a given domain.
func dnsHijackingToProxyWithHTTPURL() *TestCase {
	return &TestCase{
		Name: "dnsHijackingToProxyWithHTTPURL",
		// Disable v0.4 because it cannot detect that the DNS is consistent
		// by using the results of the TLS handshake.
		Flags: TestCaseFlagNoV04,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// add DPI rule to force all the cleartext DNS queries to
			// point the client to used the ISPProxyAddress
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{netemx.ISPProxyAddress},
				Logger:    env.Logger(),
				Domain:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         2, // StatusSuccessCleartext
			XDNSFlags:       0,
			XBlockingFlags:  32, // analysisFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}

// dnsHijackingToProxyWithHTTPSURL is the case where an ISP rule forces clients to always
// use an explicity passthrough proxy for a given domain.
func dnsHijackingToProxyWithHTTPSURL() *TestCase {
	// TODO(bassosimone): it's debateable whether this case is actually WAI but the
	// transparent TLS proxy really makes our analysis a bit more complex
	return &TestCase{
		Name: "dnsHijackingToProxyWithHTTPSURL",
		// Disable v0.4 because it cannot detect that the DNS is consistent
		// by using the results of the TLS handshake.
		Flags: TestCaseFlagNoV04,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// add DPI rule to force all the cleartext DNS queries to
			// point the client to used the ISPProxyAddress
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{netemx.ISPProxyAddress},
				Logger:    env.Logger(),
				Domain:    "www.example.com",
			})

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         1, // StatusSuccessSecure
			XDNSFlags:       0,
			XBlockingFlags:  32, // analysisFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}
