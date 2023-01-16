package portfiltering

//
// Config for the port-filtering experiment
//

import "time"

// Config contains the experiment configuration.
type Config struct {
	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `ooni:"number of milliseconds to wait before testing each port"`
}

func (c *Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return 100 * time.Millisecond
}
