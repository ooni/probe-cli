package dialer

import (
	"context"
	"errors"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/dialid"
	"github.com/ooni/probe-cli/v3/internal/multierror"
)

// DNSDialer is a dialer that uses the configured Resolver to resolver a
// domain name to IP addresses, and the configured Dialer to connect.
type DNSDialer struct {
	Dialer
	Resolver Resolver
}

// DialContext implements Dialer.DialContext.
func (d DNSDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ctx = dialid.WithDialID(ctx) // important to create before lookupHost
	var addrs []string
	addrs, err = d.LookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
	root := errors.New("address retry")
	errorunion := multierror.New(root)
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		conn, err := d.Dialer.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		errorunion.Add(err)
	}
	return nil, errorunion
}

// LookupHost implements Resolver.LookupHost
func (d DNSDialer) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.Resolver.LookupHost(ctx, hostname)
}
