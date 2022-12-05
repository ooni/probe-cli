package httpapi

//
// HTTP API descriptor (e.g., GET /api/v1/test-list/urls)
//

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
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

	// ContentType is the OPTIONAL content-type header.
	ContentType string

	// LogBody OPTIONALLY enables logging bodies.
	LogBody bool

	// MaxBodySize is the OPTIONAL maximum response body size. If
	// not set, we use the |DefaultMaxBodySize| constant.
	MaxBodySize int64

	// Method is the MANDATORY request method.
	Method string

	// RequestBody is the OPTIONAL request body.
	RequestBody []byte

	// Timeout is the OPTIONAL timeout for this call. If no timeout
	// is specified we will use the |DefaultCallTimeout| const.
	Timeout time.Duration

	// URLPath is the MANDATORY URL path.
	URLPath string

	// URLQuery is the OPTIONAL query.
	URLQuery url.Values
}

// WithBodyLogging returns a SHALLOW COPY of |Descriptor| with LogBody set to |value|. You SHOULD
// only use this method when initializing the descriptor you want to use.
func (desc *Descriptor) WithBodyLogging(value bool) *Descriptor {
	out := &Descriptor{}
	*out = *desc
	out.LogBody = value
	return out
}

// DefaultMaxBodySize is the default value for the maximum
// body size you can fetch using the httpapi package.
const DefaultMaxBodySize = 1 << 22

// DefaultCallTimeout is the default timeout for an httpapi call.
const DefaultCallTimeout = 60 * time.Second

// NewGETJSONDescriptor is a convenience factory for creating a new descriptor
// that uses the GET method and expects a JSON response.
func NewGETJSONDescriptor(urlPath string) *Descriptor {
	return NewGETJSONWithQueryDescriptor(urlPath, url.Values{})
}

// ApplicationJSON is the content-type for JSON
const ApplicationJSON = "application/json"

// NewGETJSONWithQueryDescriptor is like NewGETJSONDescriptor but it also
// allows you to provide |query| arguments. Leaving |query| nil or empty
// is equivalent to calling NewGETJSONDescriptor directly.
func NewGETJSONWithQueryDescriptor(urlPath string, query url.Values) *Descriptor {
	return &Descriptor{
		Accept:        ApplicationJSON,
		Authorization: "",
		ContentType:   "",
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       DefaultCallTimeout,
		URLPath:       urlPath,
		URLQuery:      query,
	}
}

// NewPOSTJSONWithJSONResponseDescriptor creates a descriptor that POSTs a JSON document
// and expects to receive back a JSON document from the API.
//
// This function ONLY fails if we cannot serialize the |request| to JSON. So, if you know
// that |request| is JSON-serializable, you can safely call MustNewPostJSONWithJSONResponseDescriptor instead.
func NewPOSTJSONWithJSONResponseDescriptor(urlPath string, request any) (*Descriptor, error) {
	rawRequest, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	desc := &Descriptor{
		Accept:        ApplicationJSON,
		Authorization: "",
		ContentType:   ApplicationJSON,
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   rawRequest,
		Timeout:       DefaultCallTimeout,
		URLPath:       urlPath,
		URLQuery:      nil,
	}
	return desc, nil
}

// MustNewPOSTJSONWithJSONResponseDescriptor is like NewPOSTJSONWithJSONResponseDescriptor except that
// it panics in case it's not possible to JSON serialize the |request|.
func MustNewPOSTJSONWithJSONResponseDescriptor(urlPath string, request any) *Descriptor {
	desc, err := NewPOSTJSONWithJSONResponseDescriptor(urlPath, request)
	runtimex.PanicOnError(err, "NewPOSTJSONWithJSONResponseDescriptor failed")
	return desc
}

// NewGETResourceDescriptor creates a generic descriptor for GETting a
// resource of unspecified type using the given |urlPath|.
func NewGETResourceDescriptor(urlPath string) *Descriptor {
	return &Descriptor{
		Accept:        "",
		Authorization: "",
		ContentType:   "",
		LogBody:       false,
		MaxBodySize:   DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       DefaultCallTimeout,
		URLPath:       urlPath,
		URLQuery:      url.Values{},
	}
}
