package enginenetx

import (
	"context"
	"sort"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CircoPolicy is an [HTTPSDialerPolicy] with circumvention.
//
// The zero value is not ready to use; init all the fields marked as MANDATORY.
type CircoPolicy struct {
	// Config is the MANDATORY circo root config.
	Config *CircoConfig
}

var _ HTTPSDialerPolicy = &CircoPolicy{}

// LookupTactics implements HTTPSDialerPolicy.
func (p *CircoPolicy) LookupTactics(
	ctx context.Context, domain string, reso model.Resolver) ([]HTTPSDialerTactic, error) {

	// find addresses using the DNS
	dnsAddrs, err := reso.LookupHost(ctx, domain)

	// find addresses using beacons config
	beaconExtraAddrs := p.Config.beaconsIPAddrsForDomain(domain)

	// prepare for unifying IP addrs
	unifiedAddrsMap := make(map[string]bool)
	for _, addr := range dnsAddrs {
		if !netxlite.IsBogon(addr) {
			unifiedAddrsMap[addr] = true
		}
	}
	for _, addr := range beaconExtraAddrs {
		if !netxlite.IsBogon(addr) {
			unifiedAddrsMap[addr] = true
		}
	}

	// if we don't have any address, just return the DNS lookup error
	if len(unifiedAddrsMap) < 1 {
		if err == nil {
			err = netxlite.NewErrWrapper(
				netxlite.ClassifyResolverError,
				netxlite.TopLevelOperation,
				netxlite.ErrOODNSNoAnswer,
			)
		}
		return nil, err
	}

	// sort the unified addresses for predictable testing
	sortedUnifiedAddrs := []string{}
	for addr := range unifiedAddrsMap {
		sortedUnifiedAddrs = append(sortedUnifiedAddrs, addr)
	}
	sort.Strings(sortedUnifiedAddrs) // MUTATE

	// get all the SNIs we should use
	sniv := p.Config.allServerNamesForDomainIncludingDomain(domain)
	runtimex.Assert(len(sniv) > 0, "expected at least one SNI in the SNI vector")

	// TODO(bassosimone): I am wondering whether this scheduling where
	// we always try the default SNI first is good in case there is
	// residual censorship blocking the endpoints.

	// build the tactics and make sure there is an extra delay for beacons
	// tactics such that normal tactics are used first
	//
	// regardless, make sure do do happy eyeballs with the tactics
	var (
		out          []HTTPSDialerTactic
		vanillaIndex time.Duration
		beaconsIndex time.Duration
	)
	const (
		happyEyeballsDelay = 300 * time.Millisecond
		beaconsExtraDelay  = 3 * time.Second
	)
	for _, ipAddr := range sortedUnifiedAddrs {
		for _, sni := range sniv {
			var delay time.Duration

			// Apply happy eyeballs choosing the right offset and index
			// considering whether the SNI is valid for the domain
			if sni != domain {
				delay = beaconsIndex*happyEyeballsDelay + beaconsExtraDelay
				beaconsIndex++
			} else {
				delay = vanillaIndex * happyEyeballsDelay
				vanillaIndex++

			}

			out = append(out, &circoTactic{
				Address:            ipAddr,
				TLSServerName:      sni,
				X509VerifyHostname: domain,
				InitialWaitTime:    delay,
			})
		}
	}

	// make sure policies are sorted by increasing wait time
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].InitialDelay() < out[j].InitialDelay()
	})

	return out, nil
}

// Parallelism implements HTTPSDialerPolicy.
func (p *CircoPolicy) Parallelism() int {
	return 16
}
