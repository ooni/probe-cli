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
