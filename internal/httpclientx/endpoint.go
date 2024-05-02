package httpclientx

import "github.com/ooni/probe-cli/v3/internal/model"

// Endpoint is an HTTP endpoint.
//
// The zero value is invalid; the zero value is invalid, construct using [NewEndpoint].
type Endpoint struct {
	// URL is the MANDATORY endpoint URL.
	URL string

	// Host is the OPTIONAL host header to use for cloudfronting.
	Host string
}

// NewEndpoint constructs a new [*Endpoint] instance using the given URL.
func NewEndpoint(URL string) *Endpoint {
	return &Endpoint{
		URL:  URL,
		Host: "",
	}
}

// WithHostOverride returns a copy of the [*Endpoint] using the given host header override.
func (e *Endpoint) WithHostOverride(host string) *Endpoint {
	return &Endpoint{
		URL:  e.URL,
		Host: host,
	}
}

// NewEndpointFromModelOOAPIServices constructs a new [*Endpoint] instance from the
// given [model.OOAPIService] instances, assigning the host header if "cloudfront", and
// skipping all the entries that are neither "https" not "cloudfront".
func NewEndpointFromModelOOAPIServices(svcs ...model.OOAPIService) (epnts []*Endpoint) {
	for _, svc := range svcs {
		epnt := NewEndpoint(svc.Address)
		switch svc.Type {
		case "cloudfront":
			epnt = epnt.WithHostOverride(svc.Front)
			fallthrough
		case "https":
			epnts = append(epnts, epnt)
		default:
			// skip entry
		}
	}
	return
}
