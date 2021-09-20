// Package webstepsx contains a websteps implementation
// based on the internal/measurex package.
package webstepsx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/measurex"
)

// SingleStep contains the results of a single web step.
type SingleStep struct {
	// URL is the URL this measurement refers to.
	URL string

	// LookupEndpoints contains the LookupEndpoints measurement.
	LookupEndpoints *LookupEndpoints

	// Endpoints contains all the endpoints measurements.
	Endpoints []*Endpoint
}

// BaseMeasurement is a measurement part of Result.
type BaseMeasurement struct {
	// Connect contains all the connect operations.
	Connect []*measurex.NetworkEvent

	// ReadWrite contains all the read and write operations.
	ReadWrite []*measurex.NetworkEvent

	// Close contains all the close operations.
	Close []*measurex.NetworkEvent

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*measurex.TLSHandshakeEvent

	// QUICHandshake contains all the QUIC handshakes.
	QUICHandshake []*measurex.QUICHandshakeEvent

	// LookupHost contains all the host lookups.
	LookupHost []*measurex.LookupHostEvent

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*measurex.LookupHTTPSSvcEvent

	// DNSRoundTrip contains all the DNS round trips.
	DNSRoundTrip []*measurex.DNSRoundTripEvent

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*measurex.HTTPRoundTripEvent

	// HTTPRedirect contains all the redirections.
	HTTPRedirect []*measurex.HTTPRedirectEvent
}

// LookupEndpoints describes the measurement of endpoints lookup.
type LookupEndpoints struct {
	// Domain is the domain this measurement refers to.
	Domain string

	*BaseMeasurement
}

// Endpoint describes the measurement of a given endpoint.
type Endpoint struct {
	// Endpoint is the endpoint this measurement refers to.
	Endpoint string

	*BaseMeasurement
}

// Run performs all the WebSteps step.
//
// We define "step" as the process by which we have an input URL
// and we perform the following operations:
//
// 1. lookup of all the possible endpoints for the URL;
//
// 2. measurement of each available endpoint.
//
// After a step has run, we search for all the redirection URLs
// and we run a new step with the new URLs.
//
// Arguments
//
// - ctx is the context to implement timeouts;
//
// - mx is the measurex.Measurer to use;
//
// - URL is the URL from which we start measuring;
//
// - dnsResolverUDP is the address of the DNS resolver endpoint
// using UDP we wish to use (e.g., "8.8.8.8:53").
//
// Return value
//
// A list of SingleStep structures where the Endpoints array may be empty
// if we have no been able to discover endpoints.
func Run(ctx context.Context, mx *measurex.Measurer,
	URL *url.URL, dnsResolverUDP string) (v []*SingleStep) {
	jar := measurex.NewCookieJar()
	inputs := []*url.URL{URL}
Loop:
	for len(inputs) > 0 {
		dups := make(map[string]*url.URL)
		for _, input := range inputs {
			select {
			case <-ctx.Done():
				break Loop
			default:
				mx.Infof("RunSingleStep url=%s dnsResolverUDP=%s jar=%+v",
					input, dnsResolverUDP, jar)
				m := RunSingleStep(ctx, mx, jar, input, dnsResolverUDP)
				v = append(v, m)
				for _, epnt := range m.Endpoints {
					for _, redir := range epnt.HTTPRedirect {
						dups[redir.Location.String()] = redir.Location
					}
				}
			}
		}
		inputs = nil
		for _, input := range dups {
			mx.Infof("newRedirection %s", input)
			inputs = append(inputs, input)
		}
	}
	return
}

// RunSingleStep performs a single WebSteps step.
//
// We define "step" as the process by which we have an input URL
// and we perform the following operations:
//
// 1. lookup of all the possible endpoints for the URL;
//
// 2. measurement of each available endpoint.
//
// This function DOES NOT automatically follow redirections. Though
// we have enough information to know how to follow them.
//
// Arguments
//
// - ctx is the context to implement timeouts;
//
// - mx is the measurex.Measurer to use;
//
// - cookiejar is the http.CookieJar for cookies;
//
// - URL is the URL to measure;
//
// - dnsResolverUDP is the address of the DNS resolver endpoint
// using UDP we wish to use (e.g., "8.8.8.8:53").
//
// Return value
//
// A SingleStep structure where the Endpoints array may be empty
// if we have no been able to discover endpoints.
func RunSingleStep(ctx context.Context, mx *measurex.Measurer,
	cookiekar http.CookieJar, URL *url.URL, dnsResolverUDP string) (m *SingleStep) {
	m = &SingleStep{URL: URL.String()}
	mid := mx.NewMeasurement()
	mx.Infof("LookupHTTPEndpoints measurementID=%d url=%s dnsResolverUDP=%s",
		mid, URL.String(), dnsResolverUDP)
	epnts, _ := mx.LookupHTTPEndpoints(ctx, URL, dnsResolverUDP)
	m.LookupEndpoints = &LookupEndpoints{
		Domain:          URL.Hostname(),
		BaseMeasurement: newBaseMeasurement(mx, mid),
	}
	for _, epnt := range epnts {
		mid = mx.NewMeasurement()
		mx.Infof("HTTPEndpointGet measurementID=%d url=%s endpoint=%s dnsResolverUDP=%s",
			mid, URL.String(), epnt.String(), dnsResolverUDP)
		mx.HTTPEndpointGet(ctx, epnt, cookiekar)
		m.Endpoints = append(m.Endpoints, &Endpoint{
			Endpoint:        epnt.String(),
			BaseMeasurement: newBaseMeasurement(mx, mid),
		})
	}
	return
}

// newBaseMeasurement creates a new Base Measurement.
//
// To this end, it filters all possible events by MeasurementID.
//
// Arguments
//
// - id is the MeasurementID.
//
// Return value
//
// A valid BaseMeasurement containing possibly empty lists of events.
func newBaseMeasurement(mx *measurex.Measurer, id int64) *BaseMeasurement {
	return &BaseMeasurement{
		Connect:        mx.SelectAllFromConnect(id),
		ReadWrite:      mx.SelectAllFromReadWrite(id),
		Close:          mx.SelectAllFromClose(id),
		TLSHandshake:   mx.SelectAllFromTLSHandshake(id),
		QUICHandshake:  mx.SelectAllFromQUICHandshake(id),
		LookupHost:     mx.SelectAllFromLookupHost(id),
		LookupHTTPSSvc: mx.SelectAllFromLookupHTTPSSvc(id),
		DNSRoundTrip:   mx.SelectAllFromDNSRoundTrip(id),
		HTTPRoundTrip:  mx.SelectAllFromHTTPRoundTrip(id),
		HTTPRedirect:   mx.SelectAllFromHTTPRedirect(id),
	}
}
