package torx

//
// keyvaluepair.go - key-value pair used by the control protocol.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// KeyValuePair contains a key and value pair.
type KeyValuePair struct {
	Key   string
	Value optional.Value[string]
}

// NewKeyValuePair constructs a new [*KeyValuePair].
func NewKeyValuePair(key string, value optional.Value[string]) *KeyValuePair {
	return &KeyValuePair{
		Key:   key,
		Value: value,
	}
}
