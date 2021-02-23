package ooapi

import "errors"

// Errors defined by this package. In addition to these errors, this
// package may of course return any other stdlib specific error.
var (
	ErrEmptyField      = errors.New("apiclient: empty field")
	ErrHTTPFailure     = errors.New("apiclient: http request failed")
	ErrJSONLiteralNull = errors.New("apiclient: server returned us a literal null")
	ErrMissingToken    = errors.New("apiclient: missing auth token")
	ErrUnauthorized    = errors.New("apiclient: not authorized")
)
