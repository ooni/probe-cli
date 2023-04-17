package iplookup

//
// Code to resolve the IP address using STUN
//

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/stunx"
)

// lookupSTUN performs the lookup using STUN
func (c *Client) lookupSTUN(
	ctx context.Context,
	family model.AddressFamily,
	domain, port string) (string, error) {
	// make sure we eventually time out
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// create the family aware resolver to make sure we're not going
	// to use IP addresses of the wrong family
	reso := c.newFamilyResolver(family)

	// resolve the domain name to IP addresses
	addrs, err := reso.LookupHost(ctx, domain)
	if err != nil {
		return "", err
	}

	// try each available address until one of them works
	for _, addr := range addrs {
		// create the destination endpoint
		endpoint := net.JoinHostPort(addr, port)

		// create the STUN client
		client := stunx.NewClient(endpoint, c.Logger)

		// resolve the external address
		publicAddr, err := client.LookupIPAddr(ctx)
		if err != nil {
			continue
		}

		return publicAddr, nil
	}

	return "", ErrAllEndpointsFailed
}
