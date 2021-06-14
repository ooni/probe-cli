package dialer

import (
	"context"
	"net"
)

// dnsDialer is a dialer that uses the configured Resolver to resolver a
// domain name to IP addresses, and the configured Dialer to connect.
type dnsDialer struct {
	Dialer
	Resolver Resolver
}

// DialContext implements Dialer.DialContext.
func (d *dnsDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var addrs []string
	addrs, err = d.lookupHost(ctx, onlyhost)
	if err != nil {
		return nil, err
	}
	var errorslist []error
	for _, addr := range addrs {
		target := net.JoinHostPort(addr, onlyport)
		conn, err := d.Dialer.DialContext(ctx, network, target)
		if err == nil {
			return conn, nil
		}
		errorslist = append(errorslist, err)
	}
	// TODO(kelmenhorst)
	return nil, errorslist[0]
}

// lookupHost performs a domain name resolution.
func (d *dnsDialer) lookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return d.Resolver.LookupHost(ctx, hostname)
}
