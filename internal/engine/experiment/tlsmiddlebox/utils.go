package tlsmiddlebox

//
// Utility functions for tlsmiddlebox
//

import (
	"net"
)

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
