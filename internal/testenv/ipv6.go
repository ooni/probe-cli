package testenv

import (
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// haveIPv6 is an utility function that tells us whether we have IPv6 in the
// uncensored network environments in which we run tests.
func haveIPv6() bool {
	addrs, err := net.LookupHost("dns.google")
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if v6, err := netxlite.IsIPv6(addr); err != nil || !v6 {
			continue
		}
		endpoint := net.JoinHostPort(addr, "443")
		conn, err := net.DialTimeout("tcp", endpoint, 3*time.Second)
		if conn != nil {
			conn.Close()
		}
		return err == nil
	}
	return false
}

// SupportsIPv6 indicates whether the box where we're running
// tests allows us to communicate using IPv6.
var SupportsIPv6 = haveIPv6()
