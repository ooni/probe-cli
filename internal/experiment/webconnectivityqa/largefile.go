package webconnectivityqa

import "github.com/ooni/probe-cli/v3/internal/netemx"

// largeFileWithHTTP is the case where we download a large file.
func largeFileWithHTTP() *TestCase {
	return &TestCase{
		Name:  "largeFileWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://largefile.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XBlockingFlags:  32, // AnalysisBlockingFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}

// largeFileWithHTTPS is the case where we download a large file.
func largeFileWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "largeFileWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://largefile.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XBlockingFlags:  32, // AnalysisBlockingFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}
