package enginenetx

import "time"

// happyEyeballsDelay implements an happy-eyeballs like algorithm with a
// base delay of 1 second and the given index. The index is the attempt number
// and the first attempt should have zero as its index.
//
// The standard Go library uses a 300ms delay for connecting. Because a TCP
// connect is one round trip and the TLS handshake is two round trips (roughly),
// we use 1 second as the base delay increment here.
//
// The algorithm should emit 0 as the first delay, the base delay as the
// second delay, and then it should double the base delay at each attempt,
// until we reach the 8 seconds, after which the delay increments
// linearly spacing each subsequent attempts 8 seconds in the future.
//
// By doubling the base delay, we account for the case where there are
// actual issues inside the network. By using this algorithm, we are still
// able to overlap and pack more dialing attempts overall.
func happyEyeballsDelay(idx int) time.Duration {
	const baseDelay = time.Second
	switch {
	case idx <= 0:
		return 0
	case idx == 1:
		return baseDelay
	case idx <= 4:
		return baseDelay << (idx - 1)
	default:
		return baseDelay << 3 * (time.Duration(idx) - 3)
	}
}
