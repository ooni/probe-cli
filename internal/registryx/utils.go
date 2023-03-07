package registryx

import (
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// splitPair takes in input a string in the form KEY=VALUE and splits it. This
// function returns an error if it cannot find the = character to split the string.
func splitPair(s string) (string, string, error) {
	v := strings.SplitN(s, "=", 2)
	if len(v) != 2 {
		return "", "", errors.New("invalid key-value pair")
	}
	return v[0], v[1], nil
}

// mustMakeMapStringAny makes a map from string to any using as input a list
// of key-value pairs used to initialize the map, or panics on error
func mustMakeMapStringAny(input []string) (output map[string]any) {
	output = make(map[string]any)
	for _, opt := range input {
		key, value, err := splitPair(opt)
		runtimex.PanicOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}

// mustMakeMapStringString makes a map from string to string using as input a list
// of key-value pairs used to initialize the map, or panics on error
func mustMakeMapStringString(input []string) (output map[string]string) {
	output = make(map[string]string)
	for _, opt := range input {
		key, value, err := splitPair(opt)
		runtimex.PanicOnError(err, "cannot split key-value pair")
		output[key] = value
	}
	return
}
