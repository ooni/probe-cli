package ooapi

import (
	"context"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// JSONCodec is a JSON encoder and decoder. Generally, we use a
// default JSONCodec in Client. This is the interface to implement
// if you want to override such a default.
type JSONCodec interface {
	// Encode encodes v as a serialized JSON byte slice.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes the serialized JSON byte slice into v.
	Decode(b []byte, v interface{}) error
}

// RequestMaker makes an HTTP request. Generally, we use a
// default RequestMaker in Client. This is the interface to implement
// if you want to override such a default.
type RequestMaker interface {
	// NewRequest creates a new HTTP request.
	NewRequest(ctx context.Context, method, URL string, body io.Reader) (*http.Request, error)
}

// templateExecutor parses and executes a text template.
type templateExecutor interface {
	// Execute takes in input a template string and some piece of data. It
	// returns either a string where template parameters have been replaced,
	// on success, or an error, on failure.
	Execute(tmpl string, v interface{}) (string, error)
}

// HTTPClient is the interface of a generic HTTP client. The
// stdlib's http.Client implements this interface. We use
// http.DefaultClient as the default HTTPClient used by Client.
// Consumers of this package typically provide a custom HTTPClient
// with additional functionality (e.g., DoH, circumvention).
type HTTPClient interface {
	// Do should work like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// GobCodec is a Gob encoder and decoder. Generally, we use a
// default GobCodec in Client. This is the interface to implement
// if you want to override such a default.
type GobCodec interface {
	// Encode encodes v as a serialized gob byte slice.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes the serialized gob byte slice into v.
	Decode(b []byte, v interface{}) error
}

// KVStore is a key-value store. This is the interface the
// client expect for the key-value store used to save persistent
// state (typically on the file system).
type KVStore = model.KeyValueStore
