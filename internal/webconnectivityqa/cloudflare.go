package webconnectivityqa

// cloudflareCAPTCHAWithHTTP obtains the cloudflare CAPTCHA using HTTP.
func cloudflareCAPTCHAWithHTTP() *TestCase {
	// See https://github.com/ooni/probe/issues/2661 for an explanation of why
	// here for now we're forced to declare "http-diff".
	return &TestCase{
		Name:      "cloudflareCAPTCHAWithHTTP",
		Flags:     TestCaseFlagNoV04,
		Input:     "http://www.cloudflare-cache.com/",
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
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

// cloudflareCAPTCHAWithHTTPS obtains the cloudflare CAPTCHA using HTTPS.
func cloudflareCAPTCHAWithHTTPS() *TestCase {
	return &TestCase{
		Name:      "cloudflareCAPTCHAWithHTTPS",
		Flags:     TestCaseFlagNoV04,
		Input:     "https://www.cloudflare-cache.com/",
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
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
