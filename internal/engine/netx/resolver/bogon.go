package resolver

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"0.0.0.0/8",      // "This" network (however, Linux...)
		"10.0.0.0/8",     // RFC1918
		"100.64.0.0/10",  // Carrier grade NAT
		"127.0.0.0/8",    // IPv4 loopback
		"169.254.0.0/16", // RFC3927 link-local
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"224.0.0.0/4",    // Multicast
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		runtimex.PanicOnError(err, "net.ParseCIDR failed")
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivate(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// IsBogon returns whether if an IP address is bogon. Passing to this
// function a non-IP address causes it to return bogon.
func IsBogon(address string) bool {
	ip := net.ParseIP(address)
	return ip == nil || isPrivate(ip)
}

// BogonResolver is a bogon aware resolver. When a bogon is encountered in
// a reply, this resolver will return an error.
//
// Deprecation warning
//
// This resolver is deprecated. The right thing to do would be to check
// for bogons right after a domain name resolution in the nettest.
type BogonResolver struct {
	Resolver
}

// LookupHost implements Resolver.LookupHost
func (r BogonResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	for _, addr := range addrs {
		if IsBogon(addr) {
			return nil, errorsx.ErrDNSBogon
		}
	}
	return addrs, err
}

var _ Resolver = BogonResolver{}
