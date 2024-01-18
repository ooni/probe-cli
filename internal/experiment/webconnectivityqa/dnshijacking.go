package webconnectivityqa

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// dnsHijackingToProxyWithHTTPURL is the case where an ISP rule forces clients to always
// use an explicity passthrough proxy for a given domain.
func dnsHijackingToProxyWithHTTPURL() *TestCase {
	// TODO(bassosimone): it's debateable whether this case is actually WAI but the
	// transparent TLS proxy really makes our analysis a bit more complex
	return &TestCase{
		Name:  "dnsHijackingToProxyWithHTTPURL",
		Flags: 0,
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
			DNSConsistency:  "inconsistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         2,  // StatusSuccessCleartext
			XDNSFlags:       4,  // AnalysisDNSFlagUnexpectedAddrs
			XBlockingFlags:  33, // AnalysisBlockingFlagDNSBlocking | AnalysisBlockingFlagSuccess
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
		Name:  "dnsHijackingToProxyWithHTTPSURL",
		Flags: 0,
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
			DNSConsistency:  "inconsistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         1,  // StatusSuccessSecure
			XDNSFlags:       4,  // AnalysisDNSFlagUnexpectedAddrs
			XBlockingFlags:  33, // AnalysisBlockingFlagDNSBlocking | AnalysisBlockingFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}
