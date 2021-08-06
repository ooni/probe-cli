package nwcth

import (
	"context"
	"io"
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
		jar, err = cookiejar.New(nil)
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

// discoverH3Server inspects the Alt-Svc Header of the HTTP (over TCP) response of the control measurement
// to check whether the server announces to support h3
func discoverH3Server(r *HTTPRequestMeasurement, URL *url.URL) string {
	if r == nil {
		return ""
	}
	if URL.Scheme != "https" {
		return ""
	}
	alt_svc := r.Headers.Get("Alt-Svc")
	entries := strings.Split(alt_svc, ";")
	for _, e := range entries {
		if strings.Contains(e, "h3=") {
			return "h3"
		}
		if strings.Contains(e, "h3-29=") {
			return "h3-29"
		}
	}
	return ""
}
