// Package erroror contains code to represent an error or a value.
package erroror

// Value represents an error or a value.
type Value[Type any] struct {
	Err   error
	Value Type
}
