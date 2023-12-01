package webconnectivityqa

// websiteDownNXDOMAIN describes the test case where the website domain
// is NXDOMAIN according to the TH and the probe.
func websiteDownNXDOMAIN() *TestCase {
	/*
	   TODO(bassosimone): Debateable result for v0.4, while v0.5 behaves in the
	   correct way. See <https://github.com/ooni/probe-engine/issues/579>.

	   Some historical context follows:

	   Note that MK is not doing it right here because it's suppressing the
	   dns_nxdomain_error that instead is very informative. Yet, it is reporting
	   a failure in HTTP, which miniooni does not because it does not make
	   sense to perform HTTP when there are no IP addresses.

	   The following seems indeed a bug in MK where we don't properly record the
	   actual error that occurred when performing the DNS experiment.

	   See <https://github.com/measurement-kit/measurement-kit/issues/1931>.
	*/
	return &TestCase{
		Name:      "websiteDownNXDOMAIN",
		Flags:     0,                         // see above
		Input:     "http://www.example.xyz/", // domain not defined in the simulation
		Configure: nil,
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure: "dns_nxdomain_error",
			DNSConsistency:       "consistent",
			XStatus:              2052, // StatusExperimentDNS | StatusSuccessNXDOMAIN
			XBlockingFlags:       0,
			XNullNullFlags:       1, // analysisFlagNullNullNoAddrs
			Accessible:           true,
			Blocking:             false,
		},
	}
}
