package pdsl

import "net"

// Endpoint is a string containing a TCP/UDP endpoint (e.g., 8.8.8.8:443, [::1]).
type Endpoint string

// MakeEndpointsForPort returns a [Filter] that attemps to make [Endpoint] from [IPAddr].
func MakeEndpointsForPort(port string) Filter[IPAddr, Endpoint] {
	return startFilterService(func(ipAddr IPAddr) (Endpoint, error) {
		return Endpoint(net.JoinHostPort(string(ipAddr), port)), nil
	})
}
