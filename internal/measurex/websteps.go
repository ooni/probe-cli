package measurex

import (
	"context"
	"net/url"
)

// TODO(bassosimone): we need a table for cookies since we cannot
// read them from redirects and we want an easy way to get them

// WebStepResult contains the results of Measurer.WebStep.
type WebStepResult struct {
	// URL is the URL this measurement refers to.
	URL string

	// LookupEndpoints contains the LookupEndpoints measurement.
	LookupEndpoints *WebStepLookupEndpoints

	// Endpoints contains all the endpoints measurements.
	Endpoints []*WebStepEndpoint
}

// WebStepBaseMeasurement is a measurement part of WebStepResult.
type WebStepBaseMeasurement struct {
	// Connect contains all the connect operations.
	Connect []*NetworkEvent

	// ReadWrite contains all the read and write operations.
	ReadWrite []*NetworkEvent

	// Close contains all the close operations.
	Close []*NetworkEvent

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*TLSHandshakeEvent

	// LookupHost contains all the host lookups.
	LookupHost []*LookupHostEvent

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*LookupHTTPSSvcEvent

	// DNSRoundTrip contains all the DNS round trips.
	DNSRoundTrip []*DNSRoundTripEvent

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*HTTPRoundTripEvent

	// HTTPRedirect contains all the redirections.
	HTTPRedirect []*HTTPRedirectEvent
}

// WebStepLookupEndpoints describes the measurement of endpoints lookup.
type WebStepLookupEndpoints struct {
	// Domain is the domain this measurement refers to.
	Domain string

	*WebStepBaseMeasurement
}

// WebStepEndpoint describes the measurement of a given endpoint.
type WebStepEndpoint struct {
	// Endpoint is the endpoint this measurement refers to.
	Endpoint string

	*WebStepBaseMeasurement
}

// WebStep performs a simplified WebStep measurement.
//
// We define WebStep as the process by which we have an input URL
// and we perform the following operations:
//
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
// - URL is the URL to measure;
//
// - dnsResolverUDP is the address of the DNS resolver endpoint
// using UDP we wish to use (e.g., "8.8.8.8:53").
//
// Return value
//
// A WebStepResult structure where the Endpoints array may be
// empty if we have no been able to discover endpoints.
func (mx *Measurer) WebStep(
	ctx context.Context, URL *url.URL, dnsResolverUDP string) (m *WebStepResult) {
	m = &WebStepResult{
		URL: URL.String(),
	}
	mid := mx.NewMeasurement()
	epnts, _ := mx.LookupHTTPEndpoints(ctx, URL, dnsResolverUDP)
	m.LookupEndpoints = &WebStepLookupEndpoints{
		Domain:                 URL.Hostname(),
		WebStepBaseMeasurement: mx.newWebStepBaseMeasurement(mid),
	}
	for _, epnt := range epnts {
		mid = mx.NewMeasurement()
		mx.HTTPEndpointGet(ctx, epnt)
		m.Endpoints = append(m.Endpoints, &WebStepEndpoint{
			Endpoint:               epnt.String(),
			WebStepBaseMeasurement: mx.newWebStepBaseMeasurement(mid),
		})
	}
	return
}

// newWebStepMeasurements creates a new WebStepMeasurement.
//
// To this end, it filters all possible events by MeasurementID.
//
// Arguments
//
// - id is the MeasurementID.
//
// Return value
//
// A valid WebStepMeasurement containing possibly empty lists of events.
func (mx *Measurer) newWebStepBaseMeasurement(id int64) *WebStepBaseMeasurement {
	return &WebStepBaseMeasurement{
		Connect:        mx.selectAllFromConnect(id),
		ReadWrite:      mx.selectAllFromReadWrite(id),
		Close:          mx.selectAllFromClose(id),
		TLSHandshake:   mx.selectAllFromTLSHandshake(id),
		LookupHost:     mx.selectAllFromLookupHost(id),
		LookupHTTPSSvc: mx.selectAllFromLookupHTTPSSvc(id),
		DNSRoundTrip:   mx.selectAllFromDNSRoundTrip(id),
		HTTPRoundTrip:  mx.selectAllFromHTTPRoundTrip(id),
		HTTPRedirect:   mx.selectAllFromHTTPRedirect(id),
	}
}

// selectAllFromConnect selects all the entries inside of the
// Connect table that have the given MeasurementID.
//
// Arguments
//
// - id is the MeasurementID to filter for.
//
// Return value
//
// A possibly-empty list of events.
func (mx *Measurer) selectAllFromConnect(id int64) (out []*NetworkEvent) {
	for _, ev := range mx.DB.SelectAllFromDial() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromReadWrite is like selectAllFromConnect except
// that it works on the table named ReadWrite.
func (mx *Measurer) selectAllFromReadWrite(id int64) (out []*NetworkEvent) {
	for _, ev := range mx.DB.SelectAllFromReadWrite() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromClose is like selectAllFromConnect except
// that it works on the table named Close.
func (mx *Measurer) selectAllFromClose(id int64) (out []*NetworkEvent) {
	for _, ev := range mx.DB.SelectAllFromClose() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromTLSHandshake is like selectAllFromConnect except
// that it works on the table named TLSHandshake.
func (mx *Measurer) selectAllFromTLSHandshake(id int64) (out []*TLSHandshakeEvent) {
	for _, ev := range mx.DB.SelectAllFromTLSHandshake() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromLookupHost is like selectAllFromConnect except
// that it works on the table named LookupHost.
func (mx *Measurer) selectAllFromLookupHost(id int64) (out []*LookupHostEvent) {
	for _, ev := range mx.DB.SelectAllFromLookupHost() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromLookupHTTPSSvc is like selectAllFromConnect except
// that it works on the table named LookupHTTPSSvc.
func (mx *Measurer) selectAllFromLookupHTTPSSvc(id int64) (out []*LookupHTTPSSvcEvent) {
	for _, ev := range mx.DB.SelectAllFromLookupHTTPSSvc() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromDNSRoundTrip is like selectAllFromConnect except
// that it works on the table named DNSRoundTrip.
func (mx *Measurer) selectAllFromDNSRoundTrip(id int64) (out []*DNSRoundTripEvent) {
	for _, ev := range mx.DB.SelectAllFromDNSRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromHTTPRoundTrip is like selectAllFromConnect except
// that it works on the table named HTTPRoundTrip.
func (mx *Measurer) selectAllFromHTTPRoundTrip(id int64) (out []*HTTPRoundTripEvent) {
	for _, ev := range mx.DB.SelectAllFromHTTPRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// selectAllFromHTTPRedirect is like selectAllFromConnect except
// that it works on the table named HTTPRedirect.
func (mx *Measurer) selectAllFromHTTPRedirect(id int64) (out []*HTTPRedirectEvent) {
	for _, ev := range mx.DB.SelectAllFromHTTPRedirect() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}
