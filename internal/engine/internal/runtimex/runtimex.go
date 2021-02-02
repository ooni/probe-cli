// Package runtimex contains runtime extensions. This package is inspired to the excellent
// github.com/m-lab/rtx package, except that it's simpler.
package runtimex

import "fmt"

// PanicOnError panics if err is not nil.
func PanicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}
