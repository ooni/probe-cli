// Package experimentname contains code to manipulate experiment names.
package experimentname

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/strcasex"
)

// Name is the parsed representation of an experiment name using the
// "base@version" canonical syntax
type Name struct {
	// Base is the canonical base experiment name.
	Base string

	// Version is the OPTIONAL version selector after the "@".
	// An empty Version means the default/unversioned experiment.
	Version string
}

// String returns the canonical string representation of the name: just the
// base name when there is no version, or "base@version" otherwise.
func (n Name) String() string {
	if n.Version == "" {
		return n.Base
	}
	return n.Base + "@" + n.Version
}

// Parse parses an experiment name that may use the "base@version" syntax.
//
// It splits on the first "@" and canonicalizes ONLY the base name, keeping the
// version selector verbatim.
func Parse(raw string) Name {
	base, version, found := strings.Cut(raw, "@")
	name := Name{Base: canonicalizeName(base)}
	if found {
		// Keep the version verbatim; only the base is canonicalized.
		name.Version = version
	}
	return name
}

// canonicalizeBase snake-cases the base name and applies the historical aliases.
//
// Because we allow for uppercase experiment names for backwards compatibility
// with MK, we need to add some exceptions here when mapping (e.g., DNSCheck =>
// dnscheck).
func canonicalizeName(name string) string {
	switch name = strcasex.ToSnake(name); name {
	case "ndt_7":
		name = "ndt" // since 2020-03-18, we use ndt7 to implement ndt by default
	case "dns_check":
		name = "dnscheck"
	case "stun_reachability":
		name = "stunreachability"
	default:
	}
	return name
}
