// Package experimentname contains code to manipulate experiment names.
package experimentname

import "github.com/ooni/probe-cli/v3/internal/strcasex"

// Canonicalize allows code to provide experiment names
// in a more flexible way, where we have aliases.
//
// Because we allow for uppercase experiment names for backwards
// compatibility with MK, we need to add some exceptions here when
// mapping (e.g., DNSCheck => dnscheck).
func Canonicalize(name string) string {
	switch name = strcasex.ToSnake(name); name {
	case "ndt_7":
		name = "ndt" // since 2020-03-18, we use ndt7 to implement ndt by default
	case "dns_check":
		name = "dnscheck"
	case "stun_reachability":
		name = "stunreachability"
	case "web_connectivity@v_0_5":
		name = "web_connectivity@v0.5"
	default:
	}
	return name
}
