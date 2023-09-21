package enginenetx

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPSDialerNullPolicy is the default "null" policy where we use the default
// resolver provided to LookupTactics and we use the correct SNI.
//
// We say that this is the "null" policy because this is what you would get
// by default if you were not using any policy.
//
// This policy uses an Happy-Eyeballs-like algorithm. Dial attempts are
// staggered by 300 milliseconds and up to sixteen dial attempts could be
// active at the same time. Further dials will run once one of the
// sixteen active concurrent dials have failed to connect.
type HTTPSDialerNullPolicy struct{}

var _ HTTPSDialerPolicy = &HTTPSDialerNullPolicy{}

// LookupTactics implements HTTPSDialerPolicy.
func (*HTTPSDialerNullPolicy) LookupTactics(
	ctx context.Context, domain, port string, reso model.Resolver) ([]*HTTPSDialerTactic, error) {
	addrs, err := reso.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}

	const delay = 300 * time.Millisecond
	var tactics []*HTTPSDialerTactic
	for idx, addr := range addrs {
		tactics = append(tactics, &HTTPSDialerTactic{
			Endpoint:       net.JoinHostPort(addr, port),
			InitialDelay:   time.Duration(idx) * delay, // zero for the first dial
			SNI:            domain,
			VerifyHostname: domain,
		})
	}

	return tactics, nil
}

// Parallelism implements HTTPSDialerPolicy.
func (*HTTPSDialerNullPolicy) Parallelism() int {
	return 16
}

// HTTPSDialerNullStatsTracker is the "null" [HTTPSDialerStatsTracker].
type HTTPSDialerNullStatsTracker struct{}

var _ HTTPSDialerStatsTracker = &HTTPSDialerNullStatsTracker{}

// OnStarting implements HTTPSDialerStatsTracker.
func (*HTTPSDialerNullStatsTracker) OnStarting(tactic *HTTPSDialerTactic) {
	// nothing
}

// OnSuccess implements HTTPSDialerStatsTracker.
func (*HTTPSDialerNullStatsTracker) OnSuccess(tactic *HTTPSDialerTactic) {
	// nothing
}

// OnTCPConnectError implements HTTPSDialerStatsTracker.
func (*HTTPSDialerNullStatsTracker) OnTCPConnectError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSHandshakeError implements HTTPSDialerStatsTracker.
func (*HTTPSDialerNullStatsTracker) OnTLSHandshakeError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// nothing
}

// OnTLSVerifyError implements HTTPSDialerStatsTracker.
func (*HTTPSDialerNullStatsTracker) OnTLSVerifyError(ctz context.Context, tactic *HTTPSDialerTactic, err error) {
	// nothing
}
