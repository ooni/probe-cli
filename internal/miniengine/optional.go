package miniengine

//
// Internally-used optional type
//

import "github.com/ooni/probe-cli/v3/internal/runtimex"

// optional is an optional container.
type optional[Type any] struct {
	v *Type
}

// some creates an initialized optional instance.
func some[Type any](v Type) optional[Type] {
	return optional[Type]{
		v: &v,
	}
}

// none creates an empty optional instance.
func none[Type any]() optional[Type] {
	return optional[Type]{
		v: nil,
	}
}

// IsNone returns whether the optional is empty.
func (o *optional[Type]) IsNone() bool {
	return o.v == nil
}

// Unwrap returns the optional value.
func (o *optional[Type]) Unwrap() Type {
	runtimex.Assert(!o.IsNone(), "optional[Type] is none")
	return *o.v
}
