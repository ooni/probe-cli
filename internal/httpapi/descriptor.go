package httpapi

//
// HTTP API descriptor (e.g., GET /api/v1/test-list/urls)
//

import (
	"net/url"
	"time"
)

// Descriptor contains the parameters for calling a given HTTP
// API (e.g., GET /api/v1/test-list/urls).
//
// The zero value of this struct is invalid. Please, fill all the
// fields marked as MANDATORY for correct initialization.
type Descriptor struct {
	// Accept contains the OPTIONAL accept header.
	Accept string

	// Authorization is the OPTIONAL authorization.
	Authorization string

	// AcceptEncodingGzip OPTIONALLY accepts gzip-encoding bodies.
	AcceptEncodingGzip bool

	// ContentType is the OPTIONAL content-type header.
	ContentType string

	// LogBody OPTIONALLY enables logging bodies.
	LogBody bool

	// MaxBodySize is the OPTIONAL maximum response body size. If
	// not set, we use the [DefaultMaxBodySize] constant.
	MaxBodySize int64

	// Method is the MANDATORY request method.
	Method string

	// RequestBody is the OPTIONAL request body.
	RequestBody []byte

	// Timeout is the OPTIONAL timeout for this call. If no timeout
	// is specified we will use the [DefaultCallTimeout] const.
	Timeout time.Duration

	// URLPath is the MANDATORY URL path.
	URLPath string

	// URLQuery is the OPTIONAL query.
	URLQuery url.Values
}

// DefaultMaxBodySize is the default value for the maximum
// body size you can fetch using the httpapi package.
const DefaultMaxBodySize = 1 << 24

// DefaultCallTimeout is the default timeout for an httpapi call.
const DefaultCallTimeout = 60 * time.Second

// ApplicationJSON is the content-type for JSON
const ApplicationJSON = "application/json"
