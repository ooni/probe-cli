package httpapi

//
// HTTP API Endpoint (e.g., https://api.ooni.io)
//

import "github.com/ooni/probe-cli/v3/internal/model"

// Endpoint models an HTTP endpoint on which you can call
// several HTTP APIs (e.g., https://api.ooni.io) using a
// given HTTP client potentially using a circumvention tunnel
// mechanism such as psiphon or torsf.
//
// The zero value of this struct is invalid. Please, fill all the
// fields marked as MANDATORY for correct initialization.
type Endpoint struct {
	// BaseURL is the MANDATORY endpoint base URL. We will honour the
	// path of this URL and prepend it to the actual path specified inside
	// a |Descriptor.URLPath|. However, we will always discard any query
	// that may have been set inside the BaseURL. The only query string
	// will be composed from the |Descriptor.URLQuery| values.
	//
	// For example, https://api.ooni.io.
	BaseURL string

	// HTTPClient is the MANDATORY HTTP client to use.
	//
	// For example, http.DefaultClient. You can introduce circumvention
	// here by using an HTTPClient bound to a specific tunnel.
	HTTPClient model.HTTPClient

	// Host is the OPTIONAL host header to use.
	//
	// If this field is empty we use the BaseURL's hostname. A specific
	// host header may be needed when using cloudfronting.
	Host string

	// User-Agent is the OPTIONAL user-agent to use. If empty,
	// we'll use the stdlib's default user-agent string.
	UserAgent string
}

// NewEndpointList constructs a list of API endpoints from |services|
// returned by the OONI backend (or known in advance).
//
// Arguments:
//
// - httpClient is the HTTP client to use for accessing the endpoints;
//
// - userAgent is the user agent you would like to use;
//
// - service is the list of services gathered from the backend.
func NewEndpointList(httpClient model.HTTPClient,
	userAgent string, services ...model.OOAPIService) (out []*Endpoint) {
	for _, svc := range services {
		switch svc.Type {
		case "https":
			out = append(out, &Endpoint{
				BaseURL:    svc.Address,
				HTTPClient: httpClient,
				Host:       "",
				UserAgent:  userAgent,
			})
		case "cloudfront":
			out = append(out, &Endpoint{
				BaseURL:    svc.Address,
				HTTPClient: httpClient,
				Host:       svc.Front,
				UserAgent:  userAgent,
			})
		default:
			// nothing!
		}
	}
	return
}
