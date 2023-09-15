package testingproxy

import "testing"

// TestCase is a test case implemented by this package.
type TestCase interface {
	// Name returns the test case name.
	Name() string

	// Run runs the test case.
	Run(t *testing.T)

	// Short returns whether this is a short test.
	Short() bool
}

// AllTestCases contains all the test cases.
var AllTestCases = []TestCase{
	// host network and HTTP proxy
	WithHostNetworkHTTPProxyAndURL("http://www.example.com/"),
	WithHostNetworkHTTPProxyAndURL("https://www.example.com/"),

	// host network and HTTPS proxy
	WithHostNetworkHTTPWithTLSProxyAndURL("http://www.example.com/"),
	WithHostNetworkHTTPWithTLSProxyAndURL("https://www.example.com/"),

	// netem and HTTP proxy
	WithNetemHTTPProxyAndURL("http://www.example.com/"),
	WithNetemHTTPProxyAndURL("https://www.example.com/"),

	// netem and HTTPS proxy
	WithNetemHTTPWithTLSProxyAndURL("http://www.example.com/"),
	WithNetemHTTPWithTLSProxyAndURL("https://www.example.com/"),
}
