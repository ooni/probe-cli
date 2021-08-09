package nwcth

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPConfig configures the HTTP check.
type HTTPConfig struct {
	// Jar contains the optional cookiejar from the previous hop in a redirect chain.
	Jar http.CookieJar

	// Headers contains the optional HTTP request headers.
	Headers map[string][]string

	// Transport contains the mandatory HTTP RoundTripper object.
	Transport http.RoundTripper

	// URL contains the mandatory HTTP request URL.
	URL *url.URL
}

// ErrNoH3Location means that a server's h3 support could not be derived from Alt-Svc
var ErrNoH3Location = errors.New("no h3 server location")

// HTTPDo performs the HTTP check.
// HTTPRequestMeasurement is the data object containing the HTTP Get request measurement.
// NextLocationInfo contains information needed in case of an HTTP redirect. Nil, if no redirect occured.
func HTTPDo(ctx context.Context, config *HTTPConfig) (*HTTPRequestMeasurement, *NextLocationInfo) {
	req, err := newRequest(ctx, config.URL)
	if err != nil {
		return &HTTPRequestMeasurement{Failure: newfailure(err)}, nil
	}
	for k, vs := range config.Headers {
		switch strings.ToLower(k) {
		case "user-agent":
		case "accept":
		case "accept-language":
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

	jar := config.Jar
	if jar == nil {
		jar, err = cookiejar.New(nil) // should not fail
		runtimex.PanicOnError(err, "cookiejar.New failed")
	}
	// To know whether we need to redirect, we exploit the redirect check of the http.Client:
	// http.(*Client).do calls redirectBehavior to find out if an HTTP redirect status
	// (301, 302, 303, 307, 308) was returned. Only then it uses the CheckRedirect callback.
	// I.e., the client lands in the CheckRedirect callback, if and only if we need to redirect.
	// We use an atomic value to mark that CheckRedirect has been visited.
	shouldRedirect := &atomicx.Int64{}
	client := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			shouldRedirect.Add(1)
			return http.ErrUseLastResponse
		},
		Jar:       jar,
		Transport: config.Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return &HTTPRequestMeasurement{Failure: newfailure(err)}, nil
	}
	var httpRedirect *NextLocationInfo
	loc, err := resp.Location()
	if shouldRedirect.Load() > 0 && err == nil {
		// This line is here to fix the scheme when we're following h3 redirects. The original URL
		// scheme holds the h3 protocol we're using (h3 or h3-29). As mentioned below, we should
		// find a less tricky solution to this problem, so we can simplify the code.
		loc.Scheme = config.URL.Scheme
		httpRedirect = &NextLocationInfo{jar: jar, location: loc.String()}
	}
	defer resp.Body.Close()
	headers := http.Header{}
	for k := range resp.Header {
		headers[k] = resp.Header[k]
	}
	reader := &io.LimitedReader{R: resp.Body, N: maxAcceptableBody}
	data, err := iox.ReadAllContext(ctx, reader)
	return &HTTPRequestMeasurement{
		BodyLength: int64(len(data)),
		Failure:    newfailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
	}, httpRedirect
}

// TODO(bassosimone,kelmenhorst): stuffing the h3 protocol into the scheme, rather than using a
// separate data structure holding the h3 protocol and the new URL, leads to more complex/tricky code,
// so we should probably see whether we can avoid doing that.

// newRequest creates a new *http.Request.
// h3 URL schemes are replaced by "https", to avoid invalid-scheme-errors during HTTP GET.
func newRequest(ctx context.Context, URL *url.URL) (*http.Request, error) {
	realSchemes := map[string]string{
		"http":  "http",
		"https": "https",
		"h3":    "https",
		"h3-29": "https",
	}
	newURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	newURL.Scheme = realSchemes[URL.Scheme]
	return http.NewRequestWithContext(ctx, "GET", newURL.String(), nil)
}

type altSvcH3 struct {
	authority string
	proto     string
}

// getH3Location returns the URL of the HTTP/3 location of the server,
// or ErrNoH3Location if H3 support is not advertised in the Alt-Svc Header
func getH3Location(r *HTTPRequestMeasurement, URL *url.URL) (*url.URL, error) {
	if r == nil {
		return nil, ErrNoH3Location
	}
	if URL.Scheme != "https" {
		return nil, ErrNoH3Location
	}
	h3Svc := parseAltSvc(r, URL)
	if h3Svc == nil {
		return nil, ErrNoH3Location
	}
	quicURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	quicURL.Scheme = h3Svc.proto
	quicURL.Host = h3Svc.authority
	return quicURL, nil
}

// parseAltSvc parses the Alt-Svc HTTP header for entries advertising the use of H3
func parseAltSvc(r *HTTPRequestMeasurement, URL *url.URL) *altSvcH3 {
	// TODO(bassosimone,kelmenhorst): see if we can make this algorithm more robust.
	if r == nil {
		return nil
	}
	if URL.Scheme != "https" {
		return nil
	}
	alt_svc := r.Headers.Get("Alt-Svc")
	entries := strings.Split(alt_svc, ",")
	for _, e := range entries {
		keyvalpairs := strings.Split(e, ";")
		for _, p := range keyvalpairs {
			p = strings.Replace(p, "\"", "", -1)
			kv := strings.Split(p, "=")
			if _, ok := supportedQUICVersions[kv[0]]; ok {
				host, port, err := net.SplitHostPort(kv[1])
				runtimex.PanicOnError(err, "net.SplitHostPort failed")
				if host == "" {
					host = URL.Hostname()
				}
				authority := net.JoinHostPort(host, port)
				return &altSvcH3{authority: authority, proto: kv[0]}
			}
		}
	}
	return nil
}
