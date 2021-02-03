// Package humanizex is like dustin/go-humanize
package humanizex

import "fmt"

// SI is like dustin/go-humanize.SI
func SI(value float64, unit string) string {
	value, prefix := reduce(value)
	return fmt.Sprintf("%3.0f %s%s", value, prefix, unit)
}

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
