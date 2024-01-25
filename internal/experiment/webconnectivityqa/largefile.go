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
			// TODO(bassosimone): we should skip the body check because the body is
			// truncated but somehow we don't detect that it happens
			//
			// TODO(bassosimone): I'm not 100% sure we should say that the title
			// matches when the title is not present
			XBlockingFlags: 32, // AnalysisBlockingFlagSuccess
			Accessible:     true,
			Blocking:       false,
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
			// TODO(bassosimone): we should skip the body check because the body is
			// truncated but somehow we don't detect that it happens
			//
			// TODO(bassosimone): I'm not 100% sure we should say that the title
			// matches when the title is not present
			XBlockingFlags: 32, // AnalysisBlockingFlagSuccess
			Accessible:     true,
			Blocking:       false,
		},
	}
}
