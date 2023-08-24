package webconnectivityqa

import "github.com/ooni/probe-cli/v3/internal/netemx"

// TestCase is a test case we could run with this package.
type TestCase struct {
	// Name is the test case name
	Name string

	// Input is the input URL
	Input string

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
		tlsBlockingConnectionReset(),

		sucessWithHTTP(),
		sucessWithHTTPS(),
	}
}
