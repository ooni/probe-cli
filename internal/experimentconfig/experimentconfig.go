// Package experimentconfig contains code to manage experiments configuration.
package experimentconfig

import (
	"fmt"
	"reflect"
	"strings"
)

// TODO(bassosimone): we should probably move here all the code inside
// of registry used to serialize existing options and to set values from
// generic map[string]any types.

// DefaultOptionsSerializer serializes options for [model.ExperimentTarget]
// honouring its Options method contract:
//
// 1. we do not serialize options whose name starts with "Safe";
//
// 2. we only serialize scalar values;
//
// 3. we never serialize any zero values.
//
// This method MUST be passed a pointer to a struct. Otherwise, the return
// value will be a zero-length list (either nil or empty).
func DefaultOptionsSerializer(config any) (options []string) {
	// as documented, this method MUST be passed a struct pointer
	//
	// Implementation note: the .Elem method converts a nil
	// pointer to a zero-value pointee type.
	stval := reflect.ValueOf(config)
	if stval.Kind() != reflect.Pointer {
		return
	}
	stval = stval.Elem()
	if stval.Kind() != reflect.Struct {
		return
	}

	// obtain the structure type
	stt := stval.Type()

	// cycle through the struct fields
	for idx := 0; idx < stval.NumField(); idx++ {
		// obtain the field type and value
		fieldval, fieldtype := stval.Field(idx), stt.Field(idx)

		// make sure the field is public
		if !fieldtype.IsExported() {
			continue
		}

		// make sure the field name does not start with "Safe"
		if strings.HasPrefix(fieldtype.Name, "Safe") {
			continue
		}

		// add the field iff it's a nonzero scalar
		switch fieldval.Kind() {
		case reflect.Bool,
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Float32,
			reflect.Float64,
			reflect.String:
			if fieldval.IsZero() {
				continue
			}
			options = append(options, fmt.Sprintf("%s=%v", fieldtype.Name, fieldval.Interface()))

		default:
			// nothing
		}
	}

	return
}
