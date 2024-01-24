package webconnectivityqa

import (
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// localhostWithHTTP is the case where an ISP rule redirects us to localhost.
func localhostWithHTTP() *TestCase {
	return &TestCase{
		Name:  "localhostWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// make sure all resolvers think the correct answer is localhost
			env.ISPResolverConfig().AddRecord("www.example.com", "", "127.0.0.1")
			env.OtherResolversConfig().AddRecord("www.example.com", "", "127.0.0.1")

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency: "consistent",
			XDNSFlags:      1, // AnalysisFlagDNSBogon
		},
	}
}

// localhostWithHTTPS is the case where an ISP rule redirects us to localhost.
func localhostWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "localhostWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// make sure all resolvers think the correct answer is localhost
			env.ISPResolverConfig().AddRecord("www.example.com", "", "127.0.0.1")
			env.OtherResolversConfig().AddRecord("www.example.com", "", "127.0.0.1")

		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency: "consistent",
			XDNSFlags:      1, // AnalysisFlagDNSBogon
		},
	}
}
