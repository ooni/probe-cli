package internal

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// PrepareAddrs prepares a list of addresses for TCPConnect by adding a port suffix
func PrepareAddrs(addrs []string, port string) (out []string) {
	if port == "" {
		port = "443"
	}
	for _, addr := range addrs {
		IP6, err := netxlite.IsIPv6(addr)
		if err != nil {
			continue
		}
		if IP6 {
			addr = fmt.Sprintf("%c%s%c", '[', addr, ']')
		}
		addr = fmt.Sprintf("%s%c%s", addr, ':', port)
		out = append(out, addr)
	}
	return
}
