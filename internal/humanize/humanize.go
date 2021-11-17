// Package humanize is like dustin/go-humanize.
package humanize

import "fmt"

// SI is like dustin/go-humanize.SI but its implementation is
// specially tailored for printing download speeds.
func SI(value float64, unit string) string {
	value, prefix := reduce(value)
	return fmt.Sprintf("%6.2f %s%s", value, prefix, unit)
}

// reduce reduces value to a base value and a unit prefix. For
// example, reduce(1055) returns (1.055, "k").
func reduce(value float64) (float64, string) {
	if value < 1e03 {
		return value, " "
	}
	value /= 1e03
	if value < 1e03 {
		return value, "k"
	}
	value /= 1e03
	if value < 1e03 {
		return value, "M"
	}
	value /= 1e03
	return value, "G"
}
