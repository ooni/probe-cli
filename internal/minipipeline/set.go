package minipipeline

import (
	"encoding/json"
	"sort"
)

// Set is a set containing keys with pretty JSON serialization
// and deserialization rules and a valid zero value.
type Set[T ~string | ~int64] struct {
	state map[T]bool
}

var (
	_ json.Marshaler   = Set[int64]{}
	_ json.Unmarshaler = &Set[int64]{}
)

// NewSet creates a new set containing the given keys.
func NewSet[T ~string | ~int64](keys ...T) Set[T] {
	var sx Set[T]
	sx.Add(keys...)
	return sx
}

// Add adds the given key to the set.
func (sx *Set[T]) Add(keys ...T) {
	if sx.state == nil {
		sx.state = make(map[T]bool)
	}
	for _, key := range keys {
		sx.state[key] = true
	}
}

// Len returns the number of keys inside the set.
func (sx Set[T]) Len() int {
	return len(sx.state)
}

// Remove removes the given key from the set.
func (sx Set[T]) Remove(keys ...T) {
	for _, key := range keys {
		delete(sx.state, key)
	}
}

// Keys returns the keys.
func (sx Set[T]) Keys() []T {
	keys := []T{}
	for entry := range sx.state {
		keys = append(keys, entry)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}

// MarshalJSON implements json.Marshaler.
func (sx Set[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(sx.Keys())
}

// UnmarshalJSON implements json.Unmarshaler.
func (sx *Set[T]) UnmarshalJSON(data []byte) error {
	var keys []T
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	sx.Add(keys...)
	return nil
}

// Contains returns whether the set contains a key.
func (sx *Set[T]) Contains(key T) bool {
	_, found := sx.state[key]
	return found
}
