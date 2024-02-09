package webconnectivityqa

var successCheckers = []Checker{
	// See https://github.com/ooni/probe/issues/2674
	&ReadWriteEventsExistentialChecker{},

	// See https://github.com/ooni/probe/issues/2676
	&ClientResolverCorrectnessChecker{},
}

// successWithHTTP ensures we can successfully measure an HTTP URL.
func successWithHTTP() *TestCase {
	return &TestCase{
		Name:      "successWithHTTP",
		Flags:     0,
		Input:     "http://www.example.com/",
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &TestKeys{
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
		Checkers: successCheckers,
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
		ExpectTestKeys: &TestKeys{
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
		Checkers: successCheckers,
	}
}
