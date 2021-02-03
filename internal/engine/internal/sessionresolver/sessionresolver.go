// Package sessionresolver contains the resolver used by the session. This
// resolver uses Powerdns DoH by default and falls back on the system
// provided resolver if Powerdns DoH is not working.
package sessionresolver

import (
	"context"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// Resolver is the session resolver.
type Resolver struct {
	Primary         netx.DNSClient
	PrimaryFailure  *atomicx.Int64
	PrimaryQuery    *atomicx.Int64
	Fallback        netx.DNSClient
	FallbackFailure *atomicx.Int64
	FallbackQuery   *atomicx.Int64
}

// New creates a new session resolver.
func New(config netx.Config) *Resolver {
	primary, err := netx.NewDNSClientWithOverrides(config,
		"https://cloudflare.com/dns-query", "dns.cloudflare.com", "", "")
	runtimex.PanicOnError(err, "cannot create dns over https resolver")
	fallback, err := netx.NewDNSClient(config, "system:///")
	runtimex.PanicOnError(err, "cannot create system resolver")
	return &Resolver{
		Primary:         primary,
		PrimaryFailure:  atomicx.NewInt64(),
		PrimaryQuery:    atomicx.NewInt64(),
		Fallback:        fallback,
		FallbackFailure: atomicx.NewInt64(),
		FallbackQuery:   atomicx.NewInt64(),
	}
}

// CloseIdleConnections closes the idle connections, if any
func (r *Resolver) CloseIdleConnections() {
	r.Primary.CloseIdleConnections()
	r.Fallback.CloseIdleConnections()
}

// Stats returns stats about the session resolver.
func (r *Resolver) Stats() string {
	return fmt.Sprintf("sessionresolver: failure rate: primary: %d/%d; fallback: %d/%d",
		r.PrimaryFailure.Load(), r.PrimaryQuery.Load(),
		r.FallbackFailure.Load(), r.FallbackQuery.Load())
}

// LookupHost implements Resolver.LookupHost
func (r *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	// Algorithm similar to Firefox TRR2 mode. See:
	// https://wiki.mozilla.org/Trusted_Recursive_Resolver#DNS-over-HTTPS_Prefs_in_Firefox
	// We use a higher timeout than Firefox's timeout (1.5s) to be on the safe side
	// and therefore see to use DoH more often.
	r.PrimaryQuery.Add(1)
	trr2, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	addrs, err := r.Primary.LookupHost(trr2, hostname)
	if err != nil {
		r.PrimaryFailure.Add(1)
		r.FallbackQuery.Add(1)
		addrs, err = r.Fallback.LookupHost(ctx, hostname)
		if err != nil {
			r.FallbackFailure.Add(1)
		}
	}
	return addrs, err
}

// Network implements Resolver.Network
func (r *Resolver) Network() string {
	return "sessionresolver"
}

// Address implements Resolver.Address
func (r *Resolver) Address() string {
	return ""
}
