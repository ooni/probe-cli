package webconnectivityalgo

import (
	"math/rand"
	"net"
)

// dnsOverUDPResolverAddressIPv4 is the list of DNS-over-UDP IPv4 addresses.
var dnsOverUDPResolverAddressIPv4 = []string{
	// dns.google
	"8.8.8.8",
	"8.8.4.4",

	// dns.quad9.net
	"9.9.9.9",
	"149.112.112.112",

	// cloudflare-dns.com
	"1.1.1.1",
	"1.0.0.1",

	// doh.opendns.com
	"208.67.222.222",
	"208.67.220.220",
}

// RandomDNSOverUDPResolverEndpointIPv4 returns a random DNS-over-UDP resolver endpoint using IPv4.
func RandomDNSOverUDPResolverEndpointIPv4() string {
	idx := rand.Intn(len(dnsOverUDPResolverAddressIPv4))
	return net.JoinHostPort(dnsOverUDPResolverAddressIPv4[idx], "53")
}
