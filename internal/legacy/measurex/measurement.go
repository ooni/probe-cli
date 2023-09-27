package measurex

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

//
// Measurement
//
// Here we define the fundamental measurement types
// produced by this package.
//

// URLMeasurement is the measurement of a whole URL. It contains
// a bunch of measurements detailing each measurement step.
type URLMeasurement struct {
	// URL is the URL we're measuring.
	URL string

	// DNS contains all the DNS related measurements.
	DNS []*DNSMeasurement

	// Endpoints contains a measurement for each endpoint
	// that we discovered via DNS or TH.
	Endpoints []*HTTPEndpointMeasurement

	// RedirectURLs contain the URLs to which we should fetch
	// if we choose to follow redirections.
	RedirectURLs []string

	// TH is the measurement collected by the TH. This field
	// will be nil if we cannot contact the TH.
	TH *THMeasurement

	// TotalRuntime is the total time to measure this URL.
	TotalRuntime time.Duration

	// DNSRuntime is the time to run all DNS checks.
	DNSRuntime time.Duration

	// THRuntime is the total time to invoke all test helpers.
	THRuntime time.Duration

	// EpntsRuntime is the total time to check all the endpoints.
	EpntsRuntime time.Duration
}

// fillRedirects takes in input a complete URLMeasurement and fills
// the field named Redirects with all redirections.
func (m *URLMeasurement) fillRedirects() {
	dups := make(map[string]bool)
	for _, epnt := range m.Endpoints {
		for _, redir := range epnt.HTTPRedirect {
			loc := redir.Location.String()
			if _, found := dups[loc]; found {
				continue
			}
			dups[loc] = true
			m.RedirectURLs = append(m.RedirectURLs, loc)
		}
	}
}

// Measurement groups all the events that have the same MeasurementID. This
// data format is not compatible with the OONI data format.
type Measurement struct {
	// Connect contains all the connect operations.
	Connect []*NetworkEvent

	// ReadWrite contains all the read and write operations.
	ReadWrite []*NetworkEvent

	// Close contains all the close operations.
	Close []*NetworkEvent

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*QUICTLSHandshakeEvent

	// QUICHandshake contains all the QUIC handshakes.
	QUICHandshake []*QUICTLSHandshakeEvent

	// LookupHost contains all the host lookups.
	LookupHost []*DNSLookupEvent

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*DNSLookupEvent

	// DNSRoundTrip contains all the DNS round trips.
	DNSRoundTrip []*DNSRoundTripEvent

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*HTTPRoundTripEvent

	// HTTPRedirect contains all the redirections.
	HTTPRedirect []*HTTPRedirectEvent
}

// DNSMeasurement is a DNS measurement.
type DNSMeasurement struct {
	// Domain is the domain this measurement refers to.
	Domain string

	// A DNSMeasurement is a Measurement.
	*Measurement
}

// allEndpointsForDomain returns all the endpoints for
// a specific domain contained in a measurement.
//
// Arguments:
//
// - domain is the domain we want to connect to;
//
// - port is the port for the endpoint.
func (m *DNSMeasurement) allEndpointsForDomain(domain, port string) (out []*Endpoint) {
	out = append(out, m.allTCPEndpoints(domain, port)...)
	out = append(out, m.allQUICEndpoints(domain, port)...)
	return
}

// AllEndpointsForDomain gathers all the endpoints for a given domain from
// a list of DNSMeasurements, removes duplicates and returns the result.
func AllEndpointsForDomain(domain, port string, meas ...*DNSMeasurement) ([]*Endpoint, error) {
	var out []*Endpoint
	for _, m := range meas {
		epnt := m.allEndpointsForDomain(domain, port)
		out = append(out, epnt...)
	}
	return removeDuplicateEndpoints(out...), nil
}

func (m *DNSMeasurement) allTCPEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range m.LookupHost {
		if domain != entry.Domain {
			continue
		}
		for _, addr := range entry.Addrs() {
			if net.ParseIP(addr) == nil {
				continue // skip CNAME entries courtesy the WCTH
			}
			out = append(out, m.newEndpoint(addr, port, NetworkTCP))
		}
	}
	return
}

func (m *DNSMeasurement) allQUICEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range m.LookupHTTPSSvc {
		if domain != entry.Domain {
			continue
		}
		if !entry.SupportsHTTP3() {
			continue
		}
		for _, addr := range entry.Addrs() {
			out = append(out, m.newEndpoint(addr, port, NetworkUDP))
		}
	}
	return
}

func (m *DNSMeasurement) newEndpoint(addr, port string, network EndpointNetwork) *Endpoint {
	return &Endpoint{Network: network, Address: net.JoinHostPort(addr, port)}
}

// allHTTPEndpointsForURL returns all the HTTPEndpoints matching
// a specific URL's domain inside this measurement.
//
// Arguments:
//
// - URL is the URL for which we want endpoints;
//
// - headers are the headers to use.
//
// Returns a list of endpoints or an error.
func (m *DNSMeasurement) allHTTPEndpointsForURL(
	URL *url.URL, headers http.Header) ([]*HTTPEndpoint, error) {
	domain := URL.Hostname()
	port, err := PortFromURL(URL)
	if err != nil {
		return nil, err
	}
	epnts := m.allEndpointsForDomain(domain, port)
	var out []*HTTPEndpoint
	for _, epnt := range epnts {
		if URL.Scheme != "https" && epnt.Network == NetworkUDP {
			continue // we'll only use QUIC with HTTPS
		}
		out = append(out, &HTTPEndpoint{
			Domain:  domain,
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     domain,
			ALPN:    ALPNForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  headers,
		})
	}
	return out, nil
}

// AllEndpointsForURL is like AllHTTPEndpointsForURL but return
// simple Endpoints rather than HTTPEndpoints.
func AllEndpointsForURL(URL *url.URL, meas ...*DNSMeasurement) ([]*Endpoint, error) {
	all, err := AllHTTPEndpointsForURL(URL, http.Header{}, meas...)
	if err != nil {
		return nil, err
	}
	var out []*Endpoint
	for _, epnt := range all {
		out = append(out, &Endpoint{
			Network: epnt.Network,
			Address: epnt.Address,
		})
	}
	return out, nil
}

// AllHTTPEndpointsForURL gathers all the HTTP endpoints for a given
// URL from a list of DNSMeasurements, removes duplicates and returns
// the result. This call may fail if we cannot determine the port
// from the URL, in which case we return an error. You MUST supply
// the headers you want to use for measuring.
func AllHTTPEndpointsForURL(URL *url.URL,
	headers http.Header, meas ...*DNSMeasurement) ([]*HTTPEndpoint, error) {
	var out []*HTTPEndpoint
	for _, m := range meas {
		epnt, err := m.allHTTPEndpointsForURL(URL, headers)
		if err != nil {
			return nil, err
		}
		out = append(out, epnt...)
	}
	return removeDuplicateHTTPEndpoints(out...), nil
}

// EndpointMeasurement is an endpoint measurement.
type EndpointMeasurement struct {
	// Network is the network of this endpoint.
	Network EndpointNetwork

	// Address is the address of this endpoint.
	Address string

	// An EndpointMeasurement is a Measurement.
	*Measurement
}

// HTTPEndpointMeasurement is an HTTP endpoint measurement.
type HTTPEndpointMeasurement struct {
	// URL is the URL this measurement refers to.
	URL string

	// Network is the network of this endpoint.
	Network EndpointNetwork

	// Address is the address of this endpoint.
	Address string

	// An HTTPEndpointMeasurement is a Measurement.
	*Measurement
}

// THMeasurement is the measurement performed by the TH.
type THMeasurement struct {
	// DNS contains all the DNS related measurements.
	DNS []*DNSMeasurement

	// Endpoints contains a measurement for each endpoint
	// that was discovered by the probe or the TH.
	Endpoints []*HTTPEndpointMeasurement
}
