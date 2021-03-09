package ooapi

import "errors"

// Errors defined by this package. In addition to these errors, this
// package may of course return any other stdlib specific error.
var (
	ErrAPICallFailed   = errors.New("ooapi: API call failed")
	ErrEmptyField      = errors.New("ooapi: empty field")
	ErrHTTPFailure     = errors.New("ooapi: http request failed")
	ErrJSONLiteralNull = errors.New("ooapi: server returned us a literal null")
	ErrMissingToken    = errors.New("ooapi: missing auth token")
	ErrUnauthorized    = errors.New("ooapi: not authorized")
	errCacheNotFound   = errors.New("ooapi: not found in cache")
)
