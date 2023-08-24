package webconnectivityqa

// successWithHTTP ensures we can successfully measure an HTTP URL.
func sucessWithHTTP() *TestCase {
	return &TestCase{
		Name:      "measuring http://www.example.com/ without censorship",
		Flags:     0,
		Input:     "http://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency: "consistent",
			Accessible:     true,
			Blocking:       false,
		},
	}
}

// successWithHTTPS ensures we can successfully measure an HTTPS URL.
func sucessWithHTTPS() *TestCase {
	return &TestCase{
		Name:      "measuring https://www.example.com/ without censorship",
		Flags:     0,
		Input:     "https://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSConsistency: "consistent",
			Accessible:     true,
			Blocking:       false,
		},
	}
}
