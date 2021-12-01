package oonimkall

//
// This file contains mocks for types used by tasks. Because
// we only use mocks when testing, this file is a `_test.go` file.
//

// MockableTaskEmitter is a mockable taskEmitter.
type MockableTaskEmitter struct {
	// MockableEmit allows to mock Emit.
	MockableEmit func(key string, value interface{})
}

// ensures that a MockableTaskEmitter is a taskEmitter.
var _ taskEmitter = &MockableTaskEmitter{}

// Emit implements taskEmitter.Emit.
func (e *MockableTaskEmitter) Emit(key string, value interface{}) {
	e.MockableEmit(key, value)
}
