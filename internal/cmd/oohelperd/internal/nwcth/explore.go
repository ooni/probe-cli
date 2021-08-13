package nwcth

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Explore is the second step of the test helper algorithm. Its objective
// is to enumerate all the URLs we can discover by redirection from
// the original URL in the test list. Because the test list contains by
// definition noisy data, we need this preprocessing step to learn all
// the URLs that are actually implied by the original URL.
//
// Through the explore step, we also learn about the final page on which
// we land by following the given URL. This webpage is mainly useful to
// search for block pages using the Web Connectivity algorithm.

// Explorer is the interface responsible for running Explore.
type Explorer interface {
	Explore(URL *url.URL, headers map[string][]string) ([]*RoundTrip, error)
}

// DefaultExplorer is the default Explorer.
type DefaultExplorer struct {
	resolver netxlite.Resolver
}

// RoundTrip describes a specific round trip.
type RoundTrip struct {
	// proto is the protocol used, it can be "h2", "http/1.1", "h3", "h3-29"
	proto string

	// Request is the original HTTP request. The headers
	// also include cookies.
	Request *http.Request

	// Response is the HTTP response.
	Response *http.Response

	// sortIndex is an internal field using for sorting.
	sortIndex int
}

// Explore returns a list of round trips sorted so that the first
// round trip is the first element in the list, and so on.
// Explore uses the URL and the optional headers provided by the CtrlRequest.
func (e *DefaultExplorer) Explore(URL *url.URL, headers map[string][]string) ([]*RoundTrip, error) {
	resp, err := e.get(URL, headers)
	if err != nil {
		return nil, err
	}
	rts := e.rearrange(resp, nil)
	h3URL, err := getH3URL(resp)
	if err != nil {
		return rts, nil
	}
	resp, err = e.getH3(h3URL, headers)
	if err != nil {
		return rts, nil
	}
	rts = append(rts, e.rearrange(resp, h3URL)...)
	return rts, nil
}

// rearrange takes in input the final response of an HTTP transaction and an optional h3URL
// (which is needed to derive the type of h3 protocol, i.e. h3 or h3-29),
// and produces in output a list of round trips sorted
// such that the first round trip is the first element in the out array.
func (e *DefaultExplorer) rearrange(resp *http.Response, h3URL *h3URL) (out []*RoundTrip) {
	index := 0
	for resp != nil && resp.Request != nil {
		proto := resp.Request.URL.Scheme
		if h3URL != nil {
			proto = h3URL.proto
		}
		out = append(out, &RoundTrip{
			proto:     proto,
			sortIndex: index,
			Request:   resp.Request,
			Response:  resp,
		})
		index += 1
		resp = resp.Request.Response
	}
	sh := &sortHelper{out}
	sort.Sort(sh)
	return
}

// sortHelper is the helper structure to sort round trips.
type sortHelper struct {
	v []*RoundTrip
}

// Len implements sort.Interface.Len.
func (sh *sortHelper) Len() int {
	return len(sh.v)
}

// Less implements sort.Interface.Less.
func (sh *sortHelper) Less(i, j int) bool {
	return sh.v[i].sortIndex >= sh.v[j].sortIndex
}

// Swap implements sort.Interface.Swap.
func (sh *sortHelper) Swap(i, j int) {
	sh.v[i], sh.v[j] = sh.v[j], sh.v[i]
}

// get gets the given URL and returns the final response after
// redirection, and an error. If the error is nil, the final response is valid.
func (e *DefaultExplorer) get(URL *url.URL, headers map[string][]string) (*http.Response, error) {
	tlsConf := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}
	transport := netxlite.NewHTTPTransport(NewDialerResolver(e.resolver), tlsConf, &netxlite.TLSHandshakerConfigurable{})
	jarjar, _ := cookiejar.New(nil)
	clnt := &http.Client{
		Transport: transport,
		Jar:       jarjar,
	}
	req, err := e.newRequest(URL, headers)
	runtimex.PanicOnError(err, "newRequest failed")
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp, nil
}

// getH3 uses HTTP/3 to get the given URL and returns the final
// response after redirection, and an error. If the error is nil, the final response is valid.
func (e *DefaultExplorer) getH3(h3URL *h3URL, headers map[string][]string) (*http.Response, error) {
	dialer := NewQUICDialerResolver(e.resolver)
	tlsConf := &tls.Config{
		NextProtos: []string{h3URL.proto},
	}
	transport := netxlite.NewHTTP3Transport(dialer, tlsConf)
	jarjar, _ := cookiejar.New(nil)
	clnt := &http.Client{
		Transport: transport,
		Jar:       jarjar,
	}
	req, err := e.newRequest(h3URL.URL, headers)
	runtimex.PanicOnError(err, "newRequest failed")
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp, nil
}

func (e *DefaultExplorer) newRequest(URL *url.URL, headers map[string][]string) (*http.Request, error) {
	req, err := http.NewRequest("GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range headers {
		switch strings.ToLower(k) {
		case "user-agent", "accept", "accept-language":
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	return req, nil
}
