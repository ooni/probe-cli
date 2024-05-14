package webconnectivityqa

import (
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// localhostWithHTTP is the case where the website DNS is misconfigured and returns a loopback address.
func localhostWithHTTP() *TestCase {
	return &TestCase{
		Name:  "localhostWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// make sure all resolvers think the correct answer is localhost
			runtimex.Try0(env.ISPResolverConfig().AddRecord("www.example.com", "", "127.0.0.1"))
			runtimex.Try0(env.OtherResolversConfig().AddRecord("www.example.com", "", "127.0.0.1"))

		},
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
			DNSConsistency: "consistent",
			XDNSFlags:      1, // AnalysisFlagDNSBogon
			Accessible:     false,
			Blocking:       false,
		},
	}
}

// localhostWithHTTPS is the case where the website DNS is misconfigured and returns a loopback address.
func localhostWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "localhostWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {

			// make sure all resolvers think the correct answer is localhost
			runtimex.Try0(env.ISPResolverConfig().AddRecord("www.example.com", "", "127.0.0.1"))
			runtimex.Try0(env.OtherResolversConfig().AddRecord("www.example.com", "", "127.0.0.1"))

		},
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
			DNSConsistency: "consistent",
			XDNSFlags:      1, // AnalysisFlagDNSBogon
			Accessible:     false,
			Blocking:       false,
		},
	}
}
