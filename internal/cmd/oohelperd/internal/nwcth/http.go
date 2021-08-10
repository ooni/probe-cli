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
func HTTPDo(ctx context.Context, config *HTTPConfig) *HTTPRequestMeasurement {
	req, err := newRequest(ctx, config.URL)
	if err != nil {
		return &HTTPRequestMeasurement{Failure: newfailure(err)}
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
	client := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar:       jar,
		Transport: config.Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return &HTTPRequestMeasurement{Failure: newfailure(err)}
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
	}
}

// TODO(bassosimone,kelmenhorst): stuffing the h3 protocol into the scheme, rather than using a
// separate data structure holding the h3 protocol and the new URL, leads to more complex/tricky code,
// so we should probably see whether we can avoid doing that.

// newRequest creates a new *http.Request.
// h3 URL schemes are replaced by "https", to avoid invalid-scheme-errors during HTTP GET.
func newRequest(ctx context.Context, URL *url.URL) (*http.Request, error) {
	newURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	newURL.Scheme = realSchemes[URL.Scheme]
	return http.NewRequestWithContext(ctx, "GET", newURL.String(), nil)
}

var realSchemes = map[string]string{
	"http":  "http",
	"https": "https",
	"h3":    "https",
	"h3-29": "https",
}

type altSvcH3 struct {
	authority string
	proto     string
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
