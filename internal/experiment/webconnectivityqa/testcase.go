package webconnectivityqa

import "github.com/ooni/probe-cli/v3/internal/netemx"

const (
	// TestCaseFlagNoV04 means that this test case should not be run by v0.4
	TestCaseFlagNoV04 = 1 << iota

	// TestCaseFlagNoLTE means that this test case should not be run by LTE
	TestCaseFlagNoLTE
)

// TestCase is a test case we could run with this package.
type TestCase struct {
	// Name is the test case name
	Name string

	// Flags contains binary flags describing this test case.
	Flags int64

	// Input is the input URL
	Input string

	// LongTest indicates that this is a long test.
	LongTest bool

	// Configure is an OPTIONAL hook for further configuring the scenario.
	Configure func(env *netemx.QAEnv)

	// ExpectErr is true if we expected an error
	ExpectErr bool

	// ExpectTestKeys contains the expected test keys
	ExpectTestKeys *testKeys
}

// AllTestCases returns all the defined test cases.
func AllTestCases() []*TestCase {
	return []*TestCase{
		controlFailureWithSuccessfulHTTPWebsite(),
		controlFailureWithSuccessfulHTTPSWebsite(),

		dnsBlockingAndroidDNSCacheNoData(),
		dnsBlockingNXDOMAIN(),

		dnsHijackingToProxyWithHTTPURL(),
		dnsHijackingToProxyWithHTTPSURL(),

		redirectWithConsistentDNSAndThenConnectionRefusedForHTTP(),
		redirectWithConsistentDNSAndThenConnectionRefusedForHTTPS(),
		redirectWithConsistentDNSAndThenConnectionResetForHTTP(),
		redirectWithConsistentDNSAndThenConnectionResetForHTTPS(),
		redirectWithConsistentDNSAndThenNXDOMAIN(),
		redirectWithConsistentDNSAndThenEOFForHTTP(),
		redirectWithConsistentDNSAndThenEOFForHTTPS(),
		redirectWithConsistentDNSAndThenTimeoutForHTTP(),
		redirectWithConsistentDNSAndThenTimeoutForHTTPS(),

		sucessWithHTTP(),
		sucessWithHTTPS(),

		tcpBlockingConnectTimeout(),
		tcpBlockingConnectionRefusedWithInconsistentDNS(),

		tlsBlockingConnectionResetWithConsistentDNS(),
		tlsBlockingConnectionResetWithInconsistentDNS(),

		websiteDownNXDOMAIN(),
	}
}
