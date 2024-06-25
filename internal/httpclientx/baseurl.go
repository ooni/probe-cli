package httpclientx

import "github.com/ooni/probe-cli/v3/internal/model"

// BaseURL is an HTTP-endpoint base URL.
//
// The zero value is invalid; construct using [NewBaseURL].
type BaseURL struct {
	// Value is the MANDATORY base-URL Value.
	Value string

	// HostOverride is the OPTIONAL host header to use for cloudfronting.
	HostOverride string
}

// NewBaseURL constructs a new [*BaseURL] instance using the given URL.
func NewBaseURL(URL string) *BaseURL {
	return &BaseURL{
		Value:        URL,
		HostOverride: "",
	}
}

// WithHostOverride returns a copy of the [*BaseURL] using the given host header override.
func (e *BaseURL) WithHostOverride(host string) *BaseURL {
	return &BaseURL{
		Value:        e.Value,
		HostOverride: host,
	}
}

// NewBaseURLsFromModelOOAPIServices constructs new [*BaseURL] instances from the
// given [model.OOAPIService] instances, assigning the host header if "cloudfront", and
// skipping all the entries that are neither "https" not "cloudfront".
func NewBaseURLsFromModelOOAPIServices(svcs ...model.OOAPIService) (bases []*BaseURL) {
	for _, svc := range svcs {
		base := NewBaseURL(svc.Address)
		switch svc.Type {
		case "cloudfront":
			base = base.WithHostOverride(svc.Front)
			fallthrough
		case "https":
			bases = append(bases, base)
		default:
			// skip entry
		}
	}
	return
}
