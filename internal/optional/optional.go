// Package optional contains safer code to handle optional values.
package optional

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Value is an optional value. The zero value of this structure
// is equivalent to the one you get when calling [None].
type Value[T any] struct {
	// indirect is the indirect pointer to the value.
	indirect *T
}

// None constructs an empty value.
func None[T any]() Value[T] {
	return Value[T]{nil}
}

// Some constructs a some value unless T is a pointer and points to
// nil, in which case [Some] is equivalent to [None].
func Some[T any](value T) Value[T] {
	v := Value[T]{}
	maybeSetFromValue(&v, value)
	return v
}

// maybeSetFromValue sets the underlying value unless T is a pointer
// in which case we set the Value to be empty.
func maybeSetFromValue[T any](v *Value[T], value T) {
	rv := reflect.ValueOf(value)
	if rv.Type().Kind() == reflect.Pointer && rv.IsZero() {
		v.indirect = nil
		return
	}
	v.indirect = &value
}

var _ json.Unmarshaler = &Value[int]{}

// UnmarshalJSON implements json.Unmarshaler. Note that a `null` JSON
// value always leads to an empty Value.
func (v *Value[T]) UnmarshalJSON(data []byte) error {
	// A `null` underlying value should always be equivalent to
	// invoking the None constructor of for T. While this is not
	// what the [json] package recommends doing for this case,
	// it is consistent with initializing an optional.
	if bytes.Equal(data, []byte(`null`)) {
		v.indirect = nil
		return nil
	}

	// Otherwise, let's try to unmarshal into a real value
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	// Enforce the same semantics of the Some constructor: treat
	// pointer types specially to avoid the case where we have
	// a Value that is wrapping a nil pointer but for which the
	// IsNone check actually returns false. (Maybe this check is
	// redundant but it seems better to enforce it anyway.)
	maybeSetFromValue(v, value)
	return nil
}

var _ json.Marshaler = Value[int]{}

// MarshalJSON implements json.Marshaler. An empty value serializes
// to `null` and otherwise we serialize the underluing value.
func (v Value[T]) MarshalJSON() ([]byte, error) {
	if v.indirect == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(*v.indirect)
}

// IsNone returns whether this [Value] is empty.
func (v Value[T]) IsNone() bool {
	return v.indirect == nil
}

// Unwrap returns the underlying value or panics. In case of
// panic, the value passed to panic is an error.
func (v Value[T]) Unwrap() T {
	runtimex.Assert(!v.IsNone(), "is none")
	return *v.indirect
}

// UnwrapOr returns the fallback if the [Value] is empty.
func (v Value[T]) UnwrapOr(fallback T) T {
	if v.IsNone() {
		return fallback
	}
	return v.Unwrap()
}
