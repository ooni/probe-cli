package nwcth

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
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

// RoundTrip describes a specific round trip.
type RoundTrip struct {
	// proto is the protocol used, it can be "h2", "http/1.1", "h3", "h3-29"
	proto string
	// Request is the original HTTP request. The headers
	// also include cookies. The body has already been
	// consumed but we should not be using bodies anyway.
	Request *http.Request

	// Response is the HTTP response. The body has already
	// been consumed, so you should use Body instead.
	Response *http.Response

	// sortIndex is an internal field using for sorting.
	sortIndex int
}

// Explore returns a list of round trips sorted so that the first
// round trip is the first element in the list, and so on.
func Explore(URL *url.URL) ([]*RoundTrip, error) {
	resp, err := get(URL)
	if err != nil {
		return nil, err
	}
	rts := rearrange(resp, URL.Scheme)
	if h3URL := getH3URL(resp); h3URL != nil {
		resp, err = getH3(h3URL)
		if err != nil {
			return rts, err
		}
		rts = append(rts, rearrange(resp, h3URL.proto)...)
	}
	return rts, nil
}

// rearrange takes in input the final response of an HTTP transaction and a flag
// indicating whether HTTP/3 was used, and produces in output a list of round trips sorted
// such that the first round trip is the first element in the out array.
func rearrange(resp *http.Response, proto string) (out []*RoundTrip) {
	index := 0
	for resp != nil && resp.Request != nil {
		out = append(out, &RoundTrip{
			proto:     proto,
			sortIndex: index,
			Request:   resp.Request,
			Response:  resp,
		})
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
// redirection, and an error. If the
// error is nil, the final response is valid.
func get(URL *url.URL) (*http.Response, error) {
	jarjar, _ := cookiejar.New(nil)
	clnt := &http.Client{
		Transport: http.DefaultTransport,
		Jar:       jarjar,
	}
	resp, err := clnt.Get(URL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp, nil
}

// getH3 uses HTTP/3 and gets the given URL and returns the final
// response after redirection, and an error. If the
// error is nil, the final response is valid.
func getH3(URL *h3URL) (*http.Response, error) {
	jarjar, _ := cookiejar.New(nil)
	fmt.Println(URL.proto)
	tlsconfig := &tls.Config{NextProtos: []string{URL.proto}, ServerName: URL.Hostname()}
	clnt := &http.Client{
		Transport: &http3.RoundTripper{
			Dial: func(network, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
				return quic.DialAddrEarly(addr, tlsconfig, &quic.Config{})
			},
		},
		Jar: jarjar,
	}
	resp, err := clnt.Get(URL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return resp, nil
}
