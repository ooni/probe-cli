package measurex

import (
	"errors"
	"net"
	"net/url"
)

//
// Utils
//
// This is where we put free functions.
//

// ALPNForHTTPEndpoint selects the correct ALPN for an HTTP endpoint
// given the network. On failure, we return a nil list.
func ALPNForHTTPEndpoint(network EndpointNetwork) []string {
	switch network {
	case NetworkUDP:
		return []string{"h3"}
	case NetworkTCP:
		return []string{"h2", "http/1.1"}
	default:
		return nil
	}
}

// addrStringIfNotNil returns the string of the given addr
// unless the addr is nil, in which case it returns an empty string.
func addrStringIfNotNil(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}

// ErrCannotDeterminePortFromURL indicates that we could not determine
// the correct port from the URL authority and scheme.
var ErrCannotDeterminePortFromURL = errors.New("cannot determine port from URL")

// PortFromURL returns the port determined from the URL or an error.
func PortFromURL(URL *url.URL) (string, error) {
	switch {
	case URL.Port() != "":
		return URL.Port(), nil
	case URL.Scheme == "https":
		return "443", nil
	case URL.Scheme == "http":
		return "80", nil
	default:
		return "", ErrCannotDeterminePortFromURL
	}
}

// removeDuplicateEndpoints removes duplicate endpoints from a list of endpoints.
func removeDuplicateEndpoints(epnts ...*Endpoint) (out []*Endpoint) {
	duplicates := make(map[string]*Endpoint)
	for _, epnt := range epnts {
		duplicates[epnt.String()] = epnt
	}
	for _, epnt := range duplicates {
		out = append(out, epnt)
	}
	return
}

// removeDuplicateHTTPEndpoints removes duplicate endpoints from a list of endpoints.
func removeDuplicateHTTPEndpoints(epnts ...*HTTPEndpoint) (out []*HTTPEndpoint) {
	duplicates := make(map[string]*HTTPEndpoint)
	for _, epnt := range epnts {
		duplicates[epnt.String()] = epnt
	}
	for _, epnt := range duplicates {
		out = append(out, epnt)
	}
	return
}

// HTTPEndpointsToEndpoints convers HTTPEndpoints to Endpoints
func HTTPEndpointsToEndpoints(in []*HTTPEndpoint) (out []*Endpoint) {
	for _, epnt := range in {
		out = append(out, &Endpoint{
			Network: epnt.Network,
			Address: epnt.Address,
		})
	}
	return
}
