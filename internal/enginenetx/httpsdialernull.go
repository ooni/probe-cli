package enginenetx

import (
	"context"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	ctx context.Context, domain string, reso model.Resolver) ([]HTTPSDialerTactic, error) {
	addrs, err := reso.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}

	const delay = 300 * time.Millisecond
	var tactics []HTTPSDialerTactic
	for idx, addr := range addrs {
		tactics = append(tactics, &httpsDialerNullTactic{
			Address: addr,
			Delay:   time.Duration(idx) * delay, // zero for the first dial
			Domain:  domain,
		})
	}

	return tactics, nil
}

// Parallelism implements HTTPSDialerPolicy.
func (*HTTPSDialerNullPolicy) Parallelism() int {
	return 16
}

// httpsDialerNullTactic is the default "null" tactic where we use the
// resolved IP addresses with the domain as the SNI value.
//
// We say that this is the "null" tactic because this is what you would get
// by default if you were not using any tactic.
type httpsDialerNullTactic struct {
	// Address is the IP address we resolved.
	Address string

	// Delay is the delay after which we start dialing.
	Delay time.Duration

	// Domain is the related IP address.
	Domain string
}

// IPAddr implements HTTPSDialerTactic.
func (dt *httpsDialerNullTactic) IPAddr() string {
	return dt.Address
}

// InitialDelay implements HTTPSDialerTactic.
func (dt *httpsDialerNullTactic) InitialDelay() time.Duration {
	return dt.Delay
}

// NewTLSHandshaker implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) NewTLSHandshaker(netx *netxlite.Netx, logger model.Logger) model.TLSHandshaker {
	return netx.NewTLSHandshakerStdlib(logger)
}

// OnStarting implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) OnStarting() {
	// nothing
}

// OnSuccess implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) OnSuccess() {
	// nothing
}

// OnTCPConnectError implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) OnTCPConnectError(ctx context.Context, err error) {
	// nothing
}

// OnTLSHandshakeError implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) OnTLSHandshakeError(ctx context.Context, err error) {
	// nothing
}

// OnTLSVerifyError implements HTTPSDialerTactic.
func (*httpsDialerNullTactic) OnTLSVerifyError(ctx context.Context, err error) {
	// nothing
}

// SNI implements HTTPSDialerTactic.
func (dt *httpsDialerNullTactic) SNI() string {
	return dt.Domain
}

// String implements fmt.Stringer.
func (dt *httpsDialerNullTactic) String() string {
	return fmt.Sprintf("NullTactic{Address:\"%s\" Domain:\"%s\"}", dt.Address, dt.Domain)
}

// VerifyHostname implements HTTPSDialerTactic.
func (dt *httpsDialerNullTactic) VerifyHostname() string {
	return dt.Domain
}
