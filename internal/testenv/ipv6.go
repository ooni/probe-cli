package testenv

import (
	"net"
	"strings"
	"time"
)

// haveIPv6 is an utility function that tells us whether we have IPv6 in the
// uncensored network environments in which we run tests.
func haveIPv6(
	lookup func(host string) (addrs []string, err error),
	dialTimeout func(network string, address string, timeout time.Duration) (net.Conn, error),
) bool {
	addrs, err := lookup("dns.google")
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		// Implementation note: we cannot use netxlite.IsIPv6 here because
		// netxlite tests use this function for testing
		if !strings.Contains(addr, ":") {
			continue
		}
		endpoint := net.JoinHostPort(addr, "443")
		conn, err := dialTimeout("tcp", endpoint, 3*time.Second)
		if conn != nil {
			conn.Close()
		}
		return err == nil
	}
	return false
}

// SupportsIPv6 indicates whether the box where we're running
// tests allows us to communicate using IPv6.
var SupportsIPv6 = haveIPv6(net.LookupHost, net.DialTimeout)
