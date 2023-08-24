package webconnectivityqa

// successWithHTTP ensures we can successfully measure an HTTP URL.
func sucessWithHTTP() *TestCase {
	return &TestCase{
		Name:      "measuring http://www.example.com/ without censorship",
		Input:     "http://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			Accessible: true,
			Blocking:   false,
		},
	}
}

// successWithHTTPS ensures we can successfully measure an HTTPS URL.
func sucessWithHTTPS() *TestCase {
	return &TestCase{
		Name:      "measuring https://www.example.com/ without censorship",
		Input:     "https://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			Accessible: true,
			Blocking:   false,
		},
	}
}
