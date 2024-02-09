package webconnectivityqa

// largeFileWithHTTP is the case where we download a large file.
func largeFileWithHTTP() *TestCase {
	return &TestCase{
		Name:      "largeFileWithHTTP",
		Flags:     TestCaseFlagNoV04,
		Input:     "http://largefile.com/",
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
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
		Name:      "largeFileWithHTTPS",
		Flags:     TestCaseFlagNoV04,
		Input:     "https://largefile.com/",
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
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
