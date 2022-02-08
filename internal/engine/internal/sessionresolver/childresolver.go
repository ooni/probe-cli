package sessionresolver

import (
	"context"
	"time"
)

// childResolver is the DNS client that this package uses
// to perform individual domain name resolutions.
type childResolver interface {
	// LookupHost performs a DNS lookup.
	LookupHost(ctx context.Context, domain string) ([]string, error)

	// CloseIdleConnections closes idle connections.
	CloseIdleConnections()
}

// timeLimitedLookup performs a time-limited lookup using the given re.
func (r *Resolver) timeLimitedLookup(
	ctx context.Context, re childResolver, hostname string) ([]string, error) {
	// Algorithm similar to Firefox TRR2 mode. See:
	// https://wiki.mozilla.org/Trusted_Recursive_Resolver#DNS-over-HTTPS_Prefs_in_Firefox
	// We use a higher timeout than Firefox's timeout (1.5s) to be on the safe side
	// and therefore see to use DoH more often.
	const timeout = 4 * time.Second
	return r.timeLimitedLookupx(ctx, timeout, re, hostname)
}

func (r *Resolver) timeLimitedLookupx(ctx context.Context,
	timeout time.Duration, re childResolver, hostname string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ach := make(chan []string, 1)
	errch := make(chan error, 1)
	go r.doTimeLimitedLookup(ctx, re, hostname, ach, errch)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case addrs := <-ach:
		return addrs, nil
	case err := <-errch:
		return nil, err
	}
}

func (r *Resolver) doTimeLimitedLookup(ctx context.Context, re childResolver,
	hostname string, ach chan<- []string, errch chan<- error) {
	addrs, err := re.LookupHost(ctx, hostname)
	if err != nil {
		errch <- err
		return
	}
	ach <- addrs
}
