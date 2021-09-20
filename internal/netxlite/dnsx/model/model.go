// Package model contains the dnsx model.
package model

// HTTPSSvc is an HTTPSSvc reply.
type HTTPSSvc interface {
	// ALPN returns the ALPNs inside the SVCBAlpn structure
	ALPN() []string

	// IPv4Hint returns the IPv4 hints.
	IPv4Hint() []string

	// IPv6Hint returns the IPv6 hints.
	IPv6Hint() []string
}
