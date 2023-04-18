package iplookup

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/stunx"
)

// lookupSTUNDomainPort performs the lookup using the STUN server at the given domain and port.
func (c *Client) lookupSTUNDomainPort(
	ctx context.Context, family model.AddressFamily, domain, port string) (string, error) {
	// resolve the given domain name to IP addresses
	//
	// Note: we're using an address-family aware resolver to make sure we're not
	// going to use an IP addresses belonging to the wrong family.
	addrs, err := c.familyAwareLookupHostWithImplicitTimeout(ctx, family, domain)
	if err != nil {
		return "", err
	}

	// try each available address in sequence until one of them works
	for _, addr := range addrs {
		// create the destination endpoint
		endpoint := net.JoinHostPort(addr, port)

		// resolve the external address
		publicAddr, err := c.lookupSTUNEndpoint(ctx, endpoint)
		if err != nil {
			continue
		}
		return publicAddr, nil
	}

	return "", ErrAllEndpointsFailed
}

// lookupSTUNEndpoint uses the given STUN endpoint to lookup the IP address.
func (c *Client) lookupSTUNEndpoint(ctx context.Context, endpoint string) (string, error) {
	// make sure we eventually time out
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// create client and lookup the IP address
	client := stunx.NewClient(endpoint, c.logger)
	return client.LookupIPAddr(ctx)
}
