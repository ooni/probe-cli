package richerinput

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// LoadOptions loads options from the options representation that
// is passed to richer-input-aware experiments.
func LoadOptions[T any](config *model.RicherInputConfig) (T, error) {
	var options T
	data, err := json.Marshal(config.ExtraOptions)
	if err != nil {
		return *new(T), err
	}
	if err := json.Unmarshal(data, &options); err != nil {
		return *new(T), err
	}
	return options, nil
}

// OptionsToStringList converts options to a string list. On failure
// this function returns a zero-length string list.
func OptionsToStringList[T any](options T) (output []string) {
	// obtain the config struct value
	structValue := reflect.ValueOf(options)
	if structValue.Kind() == reflect.Pointer {
		structValue = structValue.Elem()
	}
	if structValue.Kind() != reflect.Struct {
		return
	}

	// then obtain the config struct type
	structType := structValue.Type()

	// include all the possible fields
	for idx := 0; idx < structType.NumField(); idx++ {
		// obtain field value
		fieldValue := structValue.Field(idx)

		// obtain field type
		fieldType := structType.Field(idx)

		// ignore fields that are not exported
		if !fieldType.IsExported() {
			continue
		}

		// ignore fields whose name starts with "Safe"
		if strings.HasPrefix(fieldType.Name, "Safe") {
			continue
		}

		// ignore fields whose JSON tag starts with "safe_"
		if tag := fieldType.Tag.Get("json"); strings.HasPrefix(tag, "safe_") {
			continue
		}

		// ignore fields whose value is a zero value
		//
		// note: this is to avoid emitting empty fields and we generally use
		// ~zero-values according to go as defaults, so it should be okay!
		if fieldValue.IsZero() {
			continue
		}

		// ignore fields whose type is not scalar
		// TODO(bassosimone): implement

		// append the field value
		output = append(output, fmt.Sprintf("%s=%v", fieldType.Name, fieldValue.Interface()))
	}
	return
}
