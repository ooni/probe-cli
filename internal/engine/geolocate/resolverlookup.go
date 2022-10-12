package geolocate

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	// ErrNoIPAddressReturned indicates that no IP address was
	// returned by a specific DNS resolver.
	ErrNoIPAddressReturned = errors.New("geolocate: no IP address returned")
)

type dnsResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
}

type resolverLookupClient struct {
	Resolver model.Resolver
}

func (rlc resolverLookupClient) do(ctx context.Context, r dnsResolver) (string, error) {
	var ips []string
	ips, err := r.LookupHost(ctx, "whoami.v4.powerdns.org")
	if err != nil {
		return "", err
	}
	if len(ips) < 1 {
		return "", ErrNoIPAddressReturned
	}
	return ips[0], nil
}

func (rlc resolverLookupClient) LookupResolverIP(ctx context.Context) (ip string, err error) {
	return rlc.do(ctx, rlc.Resolver)
}
