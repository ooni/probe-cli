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
func (r *Resolver) timeLimitedLookup(ctx context.Context, re childResolver, hostname string) ([]string, error) {
	// Algorithm similar to Firefox TRR2 mode. See:
	// https://wiki.mozilla.org/Trusted_Recursive_Resolver#DNS-over-HTTPS_Prefs_in_Firefox
	// We use a higher timeout than Firefox's timeout (1.5s) to be on the safe side
	// and therefore see to use DoH more often.
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	return re.LookupHost(ctx, hostname)
}
