package nwcth

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPConfig configures the HTTP check.
type HTTPConfig struct {
	Jar       *cookiejar.Jar
	Headers   map[string][]string
	Transport http.RoundTripper
	URL       *url.URL
}

// HTTPDo performs the HTTP check.
func HTTPDo(ctx context.Context, config *HTTPConfig) (*CtrlHTTPRequest, *NextLocationInfo) {
	req, err := newRequest(ctx, config.URL)
	if err != nil {
		return &CtrlHTTPRequest{Failure: newfailure(err)}, nil
	}
	// The original test helper failed with extra headers while here
	// we're implementing (for now?) a more liberal approach.
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
	// redirectReq is a clone of the initial request, with possibly modified headers
	var redirectReq *http.Request

	jar := config.Jar
	if jar == nil {
		jar, err = cookiejar.New(nil)
		runtimex.PanicOnError(err, "cookiejar.New failed")
	}
	client := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			redirectReq = r
			return http.ErrUseLastResponse
		},
		Jar:       jar,
		Transport: config.Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return &CtrlHTTPRequest{Failure: newfailure(err)}, nil
	}
	var httpRedirect *NextLocationInfo = nil
	loc, _ := resp.Location()
	if loc != nil && redirectReq != nil {
		loc.Scheme = config.URL.Scheme
		httpRedirect = &NextLocationInfo{jar: jar, location: loc.String(), httpRedirectReq: redirectReq}
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: maxAcceptableBody}
	data, err := iox.ReadAllContext(ctx, reader)
	return &CtrlHTTPRequest{
		BodyLength: int64(len(data)),
		Failure:    newfailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
	}, httpRedirect
}

func newRequest(ctx context.Context, URL *url.URL) (*http.Request, error) {
	scheme := URL.Scheme
	if strings.Contains(scheme, "h3") {
		scheme = "https"
	}
	return http.NewRequestWithContext(ctx, "GET", scheme+"://"+URL.Hostname(), nil)
}

// discoverH3Server inspects the Alt-Svc Header of the HTTP (over TCP) response of the control measurement
// to check whether the server announces to support h3
func discoverH3Server(resp *CtrlHTTPMeasurement, URL *url.URL) string {
	r := resp.HTTPRequest
	if r == nil {
		return ""
	}
	if URL.Scheme != "https" {
		return ""
	}
	alt_svc := r.Headers["Alt-Svc"]
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
