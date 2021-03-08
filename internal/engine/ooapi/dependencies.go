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

// TemplateExecutor parses and executes a text template.
type TemplateExecutor interface {
	// Execute takes in input a template string and some piece of data. It
	// returns either a string where template parameters have been replaced,
	// on success, or an error, on failure.
	Execute(tmpl string, v interface{}) (string, error)
}

// HTTPClient is the interface of a generic HTTP client.
type HTTPClient interface {
	// Do should work like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// GobCodec is a Gob encoder and decoder.
type GobCodec interface {
	// Encode encodes v as a serialized gob byte slice.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes the serialized gob byte slice into v.
	Decode(b []byte, v interface{}) error
}

// KVStore is a key-value store.
type KVStore interface {
	// Get gets a value from the key-value store.
	Get(key string) ([]byte, error)

	// Set stores a value into the key-value store.
	Set(key string, value []byte) error
}
