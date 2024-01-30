package webconnectivityqa

import "github.com/ooni/probe-cli/v3/internal/netemx"

// cloudflareCAPTCHAWithHTTP obtains the cloudflare CAPTCHA using HTTP.
func cloudflareCAPTCHAWithHTTP() *TestCase {
	return &TestCase{
		Name:  "cloudflareCAPTCHAWithHTTP",
		Flags: TestCaseFlagNoV04,
		Input: "http://www.cloudflare-cache.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			StatusCodeMatch: false,
			BodyLengthMatch: false,
			BodyProportion:  0.18180740037950663,
			HeadersMatch:    true,
			TitleMatch:      false,
			XBlockingFlags:  16, // AnalysisBlockingFlagHTTPDiff
			Accessible:      false,
			Blocking:        "http-diff",
		},
	}
}

// cloudflareCAPTCHAWithHTTPS obtains the cloudflare CAPTCHA using HTTP.
func cloudflareCAPTCHAWithHTTPS() *TestCase {
	return &TestCase{
		Name:  "cloudflareCAPTCHAWithHTTPS",
		Flags: TestCaseFlagNoV04,
		Input: "https://www.cloudflare-cache.com/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			StatusCodeMatch: false,
			BodyLengthMatch: false,
			BodyProportion:  0.18180740037950663,
			HeadersMatch:    true,
			TitleMatch:      false,
			XBlockingFlags:  32, // AnalysisBlockingFlagSuccess
			Accessible:      true,
			Blocking:        false,
		},
	}
}
