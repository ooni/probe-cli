// Package dnsx contains the dnsx model.
package dnsx

// HTTPSSvc is an HTTPSSvc reply.
type HTTPSSvc struct {
	// ALPN contains the ALPNs inside the HTTPS reply
	ALPN []string

	// IPv4 contains the IPv4 hints.
	IPv4 []string

	// IPv6 contains the IPv6 hints.
	IPv6 []string
}
