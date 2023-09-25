package enginenetx

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPSDialerNullPolicy is the default "null" policy where we use the
// given resolver and the domain as the SNI.
//
// The zero value is invalid; please, init all MANDATORY fields.
//
// We say that this is the "null" policy because this is what you would get
// by default if you were not using any policy.
//
// This policy uses an Happy-Eyeballs-like algorithm. Dial attempts are
// staggered by httpsDialerHappyEyeballsDelay.
type HTTPSDialerNullPolicy struct {
	// Logger is the MANDATORY logger.
	Logger model.Logger

	// Resolver is the MANDATORY resolver.
	Resolver model.Resolver
}

var _ HTTPSDialerPolicy = &HTTPSDialerNullPolicy{}

// httpsDialerHappyEyeballsDelay is the delay after which we should start a new TCP
// connect and TLS handshake using another tactic. The standard Go library uses a 300ms
// delay for connecting. Because a TCP connect is one round trip and the TLS handshake
// is two round trips (roughly), we multiply this value by three.
const httpsDialerHappyEyeballsDelay = 900 * time.Millisecond

// LookupTactics implements HTTPSDialerPolicy.
func (p *HTTPSDialerNullPolicy) LookupTactics(
	ctx context.Context, domain, port string) <-chan *HTTPSDialerTactic {
	out := make(chan *HTTPSDialerTactic)

	go func() {
		// make sure we close the output channel when done
		defer close(out)

		// See https://github.com/ooni/probe-cli/pull/1295#issuecomment-1731243994 for context
		// on why here we MUST make sure we short-circuit IP addresses.
		resoWithShortCircuit := &netxlite.ResolverShortCircuitIPAddr{Resolver: p.Resolver}

		addrs, err := resoWithShortCircuit.LookupHost(ctx, domain)
		if err != nil {
			p.Logger.Warnf("resoWithShortCircuit.LookupHost: %s", err.Error())
			return
		}

		for idx, addr := range addrs {
			tactic := &HTTPSDialerTactic{
				Endpoint:       net.JoinHostPort(addr, port),
				InitialDelay:   happyEyeballsDelay(httpsDialerHappyEyeballsDelay, idx),
				SNI:            domain,
				VerifyHostname: domain,
			}
			out <- tactic
		}
	}()

	return out
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
func (*HTTPSDialerNullStatsTracker) OnTLSVerifyError(tactic *HTTPSDialerTactic, err error) {
	// nothing
}
