// Package ooapi contains a client for the OONI API.
//
// Other packages use this package to communicate with OONI
// background servers. Here we mainly focus on:
//
// 1. automatically generating API code from a specification
// that is composed of specially initialized Go structs.
//
// 2. adding an optional caching layer.
//
// 3. adding optional support for registration and login.
//
// 4. being able to compare our data model with the server's one.
//
// Design
//
// Most of the code in this package is auto-generated from the
// data model in ./apimodel and the definition of APIs provided
// by ./internal/generator/spec.go.
//
// We keep the generated files up-to-date by running
//
//     go generate ./...
//
// We have tests that ensure that the definition of the API
// used here is reasonably close to the server's one.
//
// Testing
//
// The following command
//
//     go test ./...
//
// will, among other things, ensure that the specs match.
//
// Running
//
//     go test -short ./...
//
// will exclude most (slow) integration tests.
//
// Architecture
//
// The ./apimodel package contains the definition of request
// and response messages. We rely on tagging to specify how
// we should encode and decode messages.
//
// The ./internal/generator contains code to generate most
// code in this package. In particular, the spec.go file is
// the specification of the APIs.
//
// - apis.go: contains the vanilla APIs.
//
// - caching.go: contains caching wrappers for every API
// that declares that it needs a cache.
//
// - callers.go: contains the Caller interfaces. A Caller
// abstracts the callable behavior of an API.
//
// - cloners.go: contains the Cloner interfaces. A Clone
// represents the possibility of cloning an existing
// API using a specific auth token.
//
// - login.go: contains wrappers allowing to implement
// registration and login for seledcted APIs.
//
// - requests.go: contains code to generate http.Requests.
//
// - responses.go: code to parse http.Responses.
package ooapi

import (
	"context"
	"io"
	"net/http"
)

// JSONCodec is a JSON encoder and decoder.
type JSONCodec interface {
	// Encode encodes v as a serialized JSON byte slice.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes the serialized JSON byte slice into v.
	Decode(b []byte, v interface{}) error
}

// RequestMaker makes an HTTP request.
type RequestMaker interface {
	// NewRequest creates a new HTTP request.
	NewRequest(ctx context.Context, method, URL string, body io.Reader) (*http.Request, error)
}

// TemplateExecutor executes a text template.
type TemplateExecutor interface {
	// Execute takes in input a template string and some piece of data. It
	// returns either a string where template parameters have been replaced,
	// on success, or an error, on failure.
	Execute(tmpl string, v interface{}) (string, error)
}

// HTTPClient is the interface of a generic HTTP client. We use this
// interface to abstract the HTTP client on which Client depends.
type HTTPClient interface {
	// Do should work like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// GobCodec is a Gob encoder and decoder.
type GobCodec interface {
	// Encode encodes v as a serialized JSON byte slice.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes the serialized JSON byte slice into v.
	Decode(b []byte, v interface{}) error
}

// KVStore is a key-value store.
type KVStore interface {
	// Get gets a value from the key-value store.
	Get(key string) ([]byte, error)

	// Set stores a value into the key-value store.
	Set(key string, value []byte) error
}
