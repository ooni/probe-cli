package websteps

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/websteps"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	utls "gitlab.com/yawning/utls.git"
)

// Explore is the second step of the test helper algorithm. Its objective
// is to enumerate all the URLs we can discover by redirection from
// the original URL in the test list. Because the test list contains by
// definition noisy data, we need this preprocessing step to learn all
// the URLs that are actually implied by the original URL.

// Explorer is the interface responsible for running Explore.
type Explorer interface {
	Explore(URL *url.URL, headers map[string][]string) ([]*RoundTrip, error)
}

// DefaultExplorer is the default Explorer.
type DefaultExplorer struct {
	resolver netxlite.Resolver
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
		// If we cannot find the HTTP/3 URL for subsequent measurements, we just continue
		// the measurement using the URLs we have found so far.
		return rts, nil
	}
	resp, err = e.getH3(h3URL, headers)
	if err != nil {
		// If we cannot follow the HTTP/3 chain, we just continue
		// the measurement using the URLs we have found so far.
		return rts, nil
	}
	rts = append(rts, e.rearrange(resp, h3URL)...)
	return rts, nil
}

// rearrange takes in input the final response of an HTTP transaction and an optional h3URL
// (which is needed to derive the type of h3 protocol, typically h3),
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
			Proto:     proto,
			SortIndex: index,
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
	return sh.v[i].SortIndex >= sh.v[j].SortIndex
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
	handshaker := &netxlite.TLSHandshakerConfigurable{
		NewConn: netxlite.NewConnUTLS(&utls.HelloChrome_Auto),
	}
	transport := websteps.NewTransportWithDialer(websteps.NewDialerResolver(e.resolver), tlsConf, handshaker)
	// TODO(bassosimone): here we should use runtimex.PanicOnError
	jarjar, _ := cookiejar.New(nil)
	clnt := &http.Client{
		Transport: transport,
		Jar:       jarjar,
	}
	// TODO(bassosimone): document why e.newRequest cannot fail.
	req, err := e.newRequest(URL, headers)
	runtimex.PanicOnError(err, "newRequest failed")
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Note that we ignore the response body.
	return resp, nil
}

// getH3 uses HTTP/3 to get the given URL and returns the final
// response after redirection, and an error. If the error is nil, the final response is valid.
func (e *DefaultExplorer) getH3(h3URL *h3URL, headers map[string][]string) (*http.Response, error) {
	dialer := websteps.NewQUICDialerResolver(e.resolver)
	tlsConf := &tls.Config{
		NextProtos: []string{h3URL.proto},
	}
	transport := netxlite.NewHTTP3Transport(dialer, tlsConf)
	// TODO(bassosimone): here we should use runtimex.PanicOnError
	jarjar, _ := cookiejar.New(nil)
	clnt := &http.Client{
		Transport: transport,
		Jar:       jarjar,
	}
	// TODO(bassosimone): document why e.newRequest cannot fail.
	req, err := e.newRequest(h3URL.URL, headers)
	runtimex.PanicOnError(err, "newRequest failed")
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	// Note that we ignore the response body.
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
