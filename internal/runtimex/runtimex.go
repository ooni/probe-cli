// Package runtimex contains runtime extensions. This package is inspired to
// https://pkg.go.dev/github.com/m-lab/go/rtx, except that it's simpler.
package runtimex

import "fmt"

// PanicOnError calls panic() if err is not nil.
func PanicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

// Assert calls panic if assertion is false.
func Assert(assertion bool, message string) {
	if !assertion {
		panic(message)
	}
}

// PanicIfTrue calls panic if assertion is true.
func PanicIfTrue(assertion bool, message string) {
	Assert(!assertion, message)
}

// PanicIfNil calls panic if the given interface is nil.
func PanicIfNil(v interface{}, message string) {
	PanicIfTrue(v == nil, message)
}
