// Package runtimex contains runtime extensions. This package is inspired to
// https://pkg.go.dev/github.com/m-lab/go/rtx, except that it's simpler.
package runtimex

import (
	"errors"
	"fmt"
)

// PanicOnError calls panic() if err is not nil. The type passed
// to panic is an error type wrapping the original error.
func PanicOnError(err error, message string) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", message, err))
	}
}

// Assert calls panic if assertion is false. The type passed to
// panic is an error constructed using errors.New(message).
func Assert(assertion bool, message string) {
	if !assertion {
		panic(errors.New(message))
	}
}

// PanicIfTrue calls panic if assertion is true. The type passed to
// panic is an error constructed using errors.New(message).
func PanicIfTrue(assertion bool, message string) {
	Assert(!assertion, message)
}

// PanicIfNil calls panic if the given interface is nil. The type passed to
// panic is an error constructed using errors.New(message).
func PanicIfNil(v any, message string) {
	PanicIfTrue(v == nil, message)
}
