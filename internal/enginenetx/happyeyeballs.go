package enginenetx

import "time"

// happyEyeballsDelay implements an happy-eyeballs like algorithm with the
// given base delay and with the given index. The index is the attempt number
// and the first attempt should have zero as its index.
//
// The algorithm should emit 0 as the first delay, the baseDelay as the
// second delay, and then it should double the base delay at each attempt,
// until we reach the 15 seconds, after which the delay increments
// linearly spacing each subsequent attempts 15 seconds in the future.
//
// By doubling the base delay, we account for the case where there are
// actual issues inside the network. By using this algorithm, we are still
// able to overlap and pack more dialing attempts overall.
func happyEyeballsDelay(baseDelay time.Duration, idx int) time.Duration {
	const cutoff = 15 * time.Second
	switch {
	case idx <= 0:
		return 0
	case idx == 1:
		return baseDelay
	default:
		delay := baseDelay
		for idx > 1 {
			switch {
			case delay < cutoff/2:
				delay *= 2
			case delay < cutoff:
				delay = cutoff
			default:
				delay += cutoff
			}
			idx -= 1
		}
		return delay
	}
}
