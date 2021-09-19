// Package model contains the dnsx model.
package model

// HTTPS is an HTTPS reply.
type HTTPS interface {
	// ALPN returns the ALPNs inside the SVCBAlpn structure
	ALPN() []string

	// IPv4Hint returns the IPv4 hints.
	IPv4Hint() []string

	// IPv6Hint returns the IPv6 hints.
	IPv6Hint() []string
}
