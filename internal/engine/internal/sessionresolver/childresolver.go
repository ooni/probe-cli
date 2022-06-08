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

// timeLimitedLookupResult is the result of a timeLimitedLookup
type timeLimitedLookupResult struct {
	addrs []string
	err   error
}

// timeLimitedLookupWithTimeout is like timeLimitedLookup but with explicit timeout.
func timeLimitedLookupWithTimeout(ctx context.Context, re model.Resolver,
	hostname string, timeout time.Duration) ([]string, error) {
	outch := make(chan *timeLimitedLookupResult, 1) // buffer
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	go func() {
		out := &timeLimitedLookupResult{}
		out.addrs, out.err = re.LookupHost(ctx, hostname)
		outch <- out
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-outch:
		return out.addrs, out.err
	}
}
