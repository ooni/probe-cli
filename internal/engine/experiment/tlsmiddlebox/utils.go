package tlsmiddlebox

//
// Utility functions for tlsmiddlebox
//

import (
	"net"
)

// prepareAddrs prepares the resolved IP addresses by
// adding the configured port as a prefix
func prepareAddrs(addrs []string, port string) (out []string) {
	if port == "" {
		port = "443"
	}
	for _, addr := range addrs {
		if net.ParseIP(addr) == nil {
			continue
		}
		addr = net.JoinHostPort(addr, port)
		out = append(out, addr)
	}
	return
}
