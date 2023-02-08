package model

import (
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// OptionalPtr is an optional pointer.
type OptionalPtr[Type any] struct {
	v *Type
}

// NewOptionalPtr creates a new OptionalPtr.
func NewOptionalPtr[Type any](v *Type) OptionalPtr[Type] {
	runtimex.Assert(v != nil, "passed nil pointer")
	return OptionalPtr[Type]{v}
}

// Unwrap returns the underlying pointer. This function panics
// if the underlying ptr is nil.
func (op OptionalPtr[Type]) Unwrap() *Type {
	runtimex.Assert(op.IsSome(), "not initialized")
	return op.v
}

// IsSome returns whether the underlying ptr is not nil.
func (op OptionalPtr[Type]) IsSome() bool {
	return op.v != nil
}

// IsNone returns whether the underlying ptr is nil.
func (op OptionalPtr[Type]) IsNone() bool {
	return !op.IsSome()
}
