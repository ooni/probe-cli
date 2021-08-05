package nwcth

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

// HTTPConfig configures the HTTP check.
type HTTPConfig struct {
	Client            *http.Client
	Headers           map[string][]string
	MaxAcceptableBody int64
	URL               string
}

// HTTPDo performs the HTTP check.
func HTTPDo(ctx context.Context, config *HTTPConfig) (*CtrlHTTPRequest, *NextLocationInfo) {
	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
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
	var redirectReq *http.Request
	config.Client.CheckRedirect = func(r *http.Request, reqs []*http.Request) error {
		redirectReq = r
		return http.ErrUseLastResponse
	}
	resp, err := config.Client.Do(req)
	if err != nil {
		return &CtrlHTTPRequest{Failure: newfailure(err)}, nil
	}
	var httpRedirect *NextLocationInfo = nil
	loc, _ := resp.Location()
	if loc != nil && redirectReq != nil {
		httpRedirect = &NextLocationInfo{location: loc.String(), httpRedirectReq: redirectReq}
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: config.MaxAcceptableBody}
	data, err := iox.ReadAllContext(ctx, reader)
	return &CtrlHTTPRequest{
		BodyLength: int64(len(data)),
		Failure:    newfailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
	}, httpRedirect
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
