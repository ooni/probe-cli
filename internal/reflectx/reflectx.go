// Package reflectx contains [reflect] extensions.
package reflectx

import (
	"reflect"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// StructOrStructPtrIsZero returns whether a given struct or struct pointer
// only contains zero value public fields. This function panic if passed a value
// that is neither a pointer to struct nor a struct. This function panics if
// passed a nil struct pointer.
func StructOrStructPtrIsZero(vop any) bool {
	vx := reflect.ValueOf(vop)
	if vx.Kind() == reflect.Pointer {
		vx = vx.Elem()
	}
	runtimex.Assert(vx.Kind() == reflect.Struct, "not a struct")
	tx := vx.Type()
	for idx := 0; idx < tx.NumField(); idx++ {
		fvx, ftx := vx.Field(idx), tx.Field(idx)
		if !ftx.IsExported() {
			continue
		}
		if !fvx.IsZero() {
			return false
		}
	}
	return true
}
