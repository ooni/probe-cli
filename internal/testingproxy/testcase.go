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

// SOCKSTestCases contains the SOCKS test cases.
var SOCKSTestCases = []TestCase{
	// with host network
	WithHostNetworkSOCKSProxyAndURL("http://www.example.com/"),
	WithHostNetworkSOCKSProxyAndURL("https://www.example.com/"),

	// with netem
	WithNetemSOCKSProxyAndURL("http://www.example.com/"),
	WithNetemSOCKSProxyAndURL("https://www.example.com/"),

	// with netem and IPv4 addresses so we test another SOCKS5 dialing mode
	WithNetemSOCKSProxyAndURL("http://93.184.216.34/"),
	WithNetemSOCKSProxyAndURL("https://93.184.216.34/"),
}

// HTTPTestCases contains the HTTP test cases.
var HTTPTestCases = []TestCase{
	// with host network and HTTP proxy
	WithHostNetworkHTTPProxyAndURL("http://www.example.com/"),
	WithHostNetworkHTTPProxyAndURL("https://www.example.com/"),

	// with host network and HTTPS proxy
	WithHostNetworkHTTPWithTLSProxyAndURL("http://www.example.com/"),
	WithHostNetworkHTTPWithTLSProxyAndURL("https://www.example.com/"),

	// with netem and HTTP proxy
	WithNetemHTTPProxyAndURL("http://www.example.com/"),
	WithNetemHTTPProxyAndURL("https://www.example.com/"),

	// with netem and HTTPS proxy
	WithNetemHTTPWithTLSProxyAndURL("http://www.example.com/"),
	WithNetemHTTPWithTLSProxyAndURL("https://www.example.com/"),
}
