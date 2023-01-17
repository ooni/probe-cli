package httpapi

//
// HTTP API descriptor (e.g., GET /api/v1/test-list/urls)
//

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// RawRequest is the type to use with [RequestDescriptor] and
// [Descriptor] when the request body is just raw bytes.
type RawRequest struct{}

// RequestDescriptor describes the request.
type RequestDescriptor[T any] struct {
	// Body is the raw request body.
	Body []byte
}

// ResponseDescriptor describes the response.
type ResponseDescriptor[T any] interface {
	// Unmarshal unmarshals the raw response into a T.
	Unmarshal(resp *http.Response, data []byte) (T, error)
}

// RawResponseDescriptor is the type to use with [Descriptor]
// when the response's body is just raw bytes.
type RawResponseDescriptor struct{}

var _ ResponseDescriptor[[]byte] = &RawResponseDescriptor{}

// Unmarshal implements ResponseDescriptor
func (r *RawResponseDescriptor) Unmarshal(resp *http.Response, data []byte) ([]byte, error) {
	return data, nil
}

// JSONResponseDescriptor is the type to use with [Descriptor]
// when the response's body is encoded using JSON.
type JSONResponseDescriptor[T any] struct{}

// Unmarshal implements ResponseDescriptor
func (r *JSONResponseDescriptor[T]) Unmarshal(resp *http.Response, data []byte) (*T, error) {
	// Important safety note: this implementation is tailored so that, when
	// the raw JSON body is `null`, we DO NOT return `nil`, `nil`. Because
	// we create a T on the stack and then let it escape, in such a case the
	// code will instead return an empty T and nil. Returning an empty T is
	// slightly better because the caller does not need to worry about the
	// returned pointer also being nil, but they just need to worry about
	// whether any field inside the returned struct is the zero value.
	//
	// (Of course, the above reasoning breaks if the caller asks for a T
	// equal to `*Foo`, which causes the return value to be `**Foo`. That
	// said, in all cases in OONI we have T equal to `Foo` and we return
	// a `*Foo` type. This scenario is, in fact, the only one making sense
	// when you're reading a JSON from a server. So, while the problem is
	// only solved for a sub-problem, this sub-problem is the one that matters.)
	//
	// Because this safety property is important, there is also a test that
	// makes sure we don't return `nil`, `nil` with `null` input.
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

// Descriptor contains the parameters for calling a given HTTP
// API (e.g., GET /api/v1/test-list/urls).
//
// The zero value of this struct is invalid. Please, fill all the
// fields marked as MANDATORY for correct initialization.
type Descriptor[RequestType, ResponseType any] struct {
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

	// Response is the MANDATORY response descriptor.
	Response ResponseDescriptor[ResponseType]

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
