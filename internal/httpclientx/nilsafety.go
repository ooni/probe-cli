package httpclientx

import (
	"errors"
	"reflect"
)

// ErrIsNil indicates that [NilSafetyErrorIfNil] was passed a nil value.
var ErrIsNil = errors.New("nil map, pointer, or slice")

// NilSafetyErrorIfNil returns [ErrIsNil] iff input is a nil map, struct, or slice.
//
// This mechanism prevents us from mistakenly sending to a server a literal JSON "null" and
// protects us from attempting to process a literal JSON "null" from a server.
func NilSafetyErrorIfNil[Type any](value Type) (Type, error) {
	switch rv := reflect.ValueOf(value); rv.Kind() {
	case reflect.Map, reflect.Pointer, reflect.Slice:
		if rv.IsNil() {
			return zeroValue[Type](), ErrIsNil
		}
	}
	return value, nil
}

// NilSafetyAvoidNilBytesSlice replaces a nil bytes slice with an empty slice.
func NilSafetyAvoidNilBytesSlice(input []byte) []byte {
	if input == nil {
		input = []byte{}
	}
	return input
}
