package measurex

//
// Bogon
//
// This file helps us to decide if an IPAddr is a bogon.
//

// TODO(bassosimone): code in engine/netx should use this file.

import (
	"net"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// isBogon returns whether if an IP address is bogon. Passing to this
// function a non-IP address causes it to return true.
func isBogon(address string) bool {
	ip := net.ParseIP(address)
	return ip == nil || isPrivate(ip)
}

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
