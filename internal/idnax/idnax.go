// Package idnax contains IDNA extensions.
package idnax

import "golang.org/x/net/idna"

// ToASCII converts an IDNA to ASCII using the [idna.Lookup] profile.
func ToASCII(domain string) (string, error) {
	return idna.Lookup.ToASCII(domain)
}
