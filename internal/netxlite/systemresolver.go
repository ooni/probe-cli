package netxlite

//
// Discovering the system resolver address
//

import (
	"net"

	"github.com/miekg/dns"
)

// getSystemResolverAddress returns the host:port address of the first
// nameserver configured in /etc/resolv.conf, if any.
func getSystemResolverAddress() (string, bool) {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || config == nil || len(config.Servers) < 1 {
		return "", false
	}
	port := config.Port
	if port == "" {
		port = "53"
	}
	return net.JoinHostPort(config.Servers[0], port), true
}
