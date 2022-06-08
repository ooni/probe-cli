package sessionresolver

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// defaultTimeLimitedLookupTimeout is the default timeout the code should
// pass to the timeLimitedLookup function.
//
// This algorithm is similar to Firefox using TRR2 mode. See:
// https://wiki.mozilla.org/Trusted_Recursive_Resolver#DNS-over-HTTPS_Prefs_in_Firefox
//
// We use a higher timeout than Firefox's timeout (1.5s) to be on the safe side
// and therefore see to use DoH more often.
const defaultTimeLimitedLookupTimeout = 4 * time.Second

// timeLimitedLookup performs a time-limited lookup using the given re.
func timeLimitedLookup(ctx context.Context, re model.Resolver, hostname string) ([]string, error) {
	return timeLimitedLookupWithTimeout(ctx, re, hostname, defaultTimeLimitedLookupTimeout)
}

// timeLimitedLookupWithTimeout is like timeLimitedLookup but with explicit timeout.
func timeLimitedLookupWithTimeout(ctx context.Context, re model.Resolver,
	hostname string, timeout time.Duration) ([]string, error) {
	// In https://github.com/ooni/probe-cli/pull/807, I modified this code to
	// run in a background goroutine and this resulted in a data race, see
	// https://github.com/ooni/probe/issues/2135#issuecomment-1149840579. While
	// I could not reproduce the data race in a simple way, the race itself
	// seems to happen inside the http3 package. For now, I am going to revert
	// the change causing the race and I'll investigate later.
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return re.LookupHost(ctx, hostname)
}
