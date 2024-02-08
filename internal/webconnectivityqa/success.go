package webconnectivityqa

// successWithHTTP ensures we can successfully measure an HTTP URL.
func successWithHTTP() *TestCase {
	return &TestCase{
		Name:      "successWithHTTP",
		Flags:     0,
		Input:     "http://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         2,
			XBlockingFlags:  32,
			Accessible:      true,
			Blocking:        false,
		},
	}
}

// successWithHTTPS ensures we can successfully measure an HTTPS URL.
func successWithHTTPS() *TestCase {
	return &TestCase{
		Name:      "successWithHTTPS",
		Flags:     0,
		Input:     "https://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency:  "consistent",
			BodyLengthMatch: true,
			BodyProportion:  1,
			StatusCodeMatch: true,
			HeadersMatch:    true,
			TitleMatch:      true,
			XStatus:         1,
			XBlockingFlags:  32,
			Accessible:      true,
			Blocking:        false,
		},
	}
}
