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
	URL string `json:"url"`

	// Oddities contains all the oddities of all endpoints.
	Oddities []measurex.Oddity `json:"oddities"`

	// DNS contains all the DNS measurements.
	DNS []*measurex.Measurement `json:"dns"`

	// Control contains all the control measurements.
	Control []*measurex.Measurement `json:"control"`

	// Endpoints contains a measurement for each endpoints (which
	// may be empty if DNS lookup failed).
	Endpoints []*measurex.Measurement `json:"endpoints"`
}

// computeOddities computes the Oddities field my merging all
// the oddities appearing in the Endpoints list.
func (ss *SingleStep) computeOddities() {
	unique := make(map[measurex.Oddity]bool)
	for _, entry := range ss.DNS {
		for _, oddity := range entry.Oddities {
			unique[oddity] = true
		}
	}
	for _, entry := range ss.Endpoints {
		for _, oddity := range entry.Oddities {
			unique[oddity] = true
		}
	}
	for oddity := range unique {
		if oddity != "" {
			ss.Oddities = append(ss.Oddities, oddity)
		}
	}
}

// URLMeasurer measures a single URL.
//
// Make sure you fill the fields marked as MANDATORY.
type URLMeasurer struct {
	// DNSResolverUDP is the MANDATORY address of an DNS
	// over UDP resolver (e.g., "8.8.4.4.:53").
	DNSResolverUDP string

	// Mx is the MANDATORY measurex.Measurer.
	Mx *measurex.Measurer

	// URL is the MANDATORY URL to measure.
	URL *url.URL
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
// Return value:
//
// A list of SingleStep structures where the Endpoints array may be empty
// if we have no been able to discover endpoints.
func (um *URLMeasurer) Run(ctx context.Context) (v []*SingleStep) {
	jar := measurex.NewCookieJar()
	inputs := []*url.URL{um.URL}
Loop:
	for len(inputs) > 0 {
		dups := make(map[string]*url.URL)
		for _, input := range inputs {
			select {
			case <-ctx.Done():
				break Loop
			default:
				um.Mx.Infof("RunSingleStep url=%s dnsResolverUDP=%s jar=%+v",
					input, um.DNSResolverUDP, jar)
				m := um.RunSingleStep(ctx, jar, input)
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
			um.Mx.Infof("newRedirection %s", input)
			inputs = append(inputs, input)
		}
	}
	return
}

// RunSingleStep performs a single WebSteps step.
//
// This function DOES NOT automatically follow redirections.
//
// Arguments:
//
// - ctx is the context to implement timeouts;
//
// - cookiejar is the http.CookieJar for cookies;
//
// - URL is the URL to measure.
//
// Return value:
//
// A SingleStep structure where the Endpoints array may be empty
// if we have no been able to discover endpoints.
func (um *URLMeasurer) RunSingleStep(ctx context.Context,
	cookiekar http.CookieJar, URL *url.URL) (m *SingleStep) {
	m = &SingleStep{URL: URL.String()}
	defer m.computeOddities()
	port, err := measurex.PortFromURL(URL)
	if err != nil {
		return
	}
	switch URL.Scheme {
	case "https":
		m.DNS = append(m.DNS, um.Mx.LookupHTTPSSvcUDP(
			ctx, URL.Hostname(), um.DNSResolverUDP))
	default:
		// nothing to do
	}
	m.DNS = append(m.DNS, um.Mx.LookupHostSystem(ctx, URL.Hostname()))
	m.DNS = append(m.DNS, um.Mx.LookupHostUDP(ctx, URL.Hostname(), um.DNSResolverUDP))
	endpoints := um.Mx.DB.SelectAllEndpointsForDomain(URL.Hostname(), port)
	m.Control = append(m.Control, um.Mx.LookupWCTH(ctx, URL, endpoints, port))
	httpEndpoints, err := um.Mx.DB.SelectAllHTTPEndpointsForURL(URL)
	if err != nil {
		return
	}
	for _, epnt := range httpEndpoints {
		m.Endpoints = append(m.Endpoints, um.Mx.HTTPEndpointGet(ctx, epnt, cookiekar))
	}
	return
}
