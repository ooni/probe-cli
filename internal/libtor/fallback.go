//go:build !ooni_libtor

package libtor

import "github.com/cretz/bine/process"

// MaybeCreator returns a valid [process.Creator], if possible, otherwise false.
func MaybeCreator() (process.Creator, bool) {
	return nil, false
}
