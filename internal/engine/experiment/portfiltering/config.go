package portfiltering

//
// Config for the port-filtering experiment
//

// Config contains the experiment configuration.
type Config struct {
	// TestHelper is the URL to use for port-scanning
	TestHelper string `ooni:"testhelper URL for port scanning"`
}

func (c *Config) testhelper() string {
	if c.TestHelper != "" {
		return c.TestHelper
	}
	return "http://127.0.0.1"
}
