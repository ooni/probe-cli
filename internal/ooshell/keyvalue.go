package ooshell

//
// keyvalue.go
//
// Code to manage key-value pairs.
//

import (
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine"
)

// addOptions adds extra options to the given builder.
func (exp *experimentDB) addOptions(builder *engine.ExperimentBuilder) error {
	options, err := newKeyValueMap(exp.env.Options)
	if err != nil {
		return err
	}
	return builder.SetOptionsGuessType(options)
}

// parseAnnotations parses annotations.
func (exp *experimentDB) parseAnnotations() (map[string]string, error) {
	return newKeyValueMap(exp.env.Annotations)
}

// newKeyValueMap converts a slice of strings having the format
// key=value to a map mapping keys to values.
func newKeyValueMap(input []string) (map[string]string, error) {
	output := make(map[string]string)
	for _, opt := range input {
		key, value, err := newKeyValuePair(opt)
		if err != nil {
			return nil, err
		}
		output[key] = value
	}
	return output, nil
}

// newKeyValuePair converts a string having the format key=value
// to a pair containing the key and the value.
func newKeyValuePair(s string) (string, string, error) {
	v := strings.SplitN(s, "=", 2)
	if len(v) != 2 {
		return "", "", errors.New("invalid key-value pair")
	}
	return v[0], v[1], nil
}
