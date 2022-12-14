package httpapi

//
// HTTP API descriptor (e.g., GET /api/v1/test-list/urls)
//

import (
	"net/url"
	"time"
)

// RawRequest is the type to use with [RequestDescriptor] and
// [Descriptor] when the request is just raw bytes.
type RawRequest struct{}

// RequestDescriptor describes the request.
type RequestDescriptor[T any] struct {
	// Body is the raw request body.
	Body []byte
}

// Descriptor contains the parameters for calling a given HTTP
// API (e.g., GET /api/v1/test-list/urls).
//
// The zero value of this struct is invalid. Please, fill all the
// fields marked as MANDATORY for correct initialization.
type Descriptor[RequestType any] struct {
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

	// Request is the OPTIONAL request descriptor.
	Request *RequestDescriptor[RequestType]

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
