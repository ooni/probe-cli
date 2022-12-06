package main

//
// HTTP measurements
//

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// TODO(bassosimone): we should refactor the TH to use step-by-step such that we
// can use an existing connection for the HTTP-measuring task

// ctrlHTTPResponse is the result of the HTTP check performed by
// the Web Connectivity test helper.
type ctrlHTTPResponse = model.THHTTPRequestResult

// httpConfig configures the HTTP check.
type httpConfig struct {
	// Headers is OPTIONAL and contains the request headers we should set.
	Headers map[string][]string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MaxAcceptableBody is MANDATORY and specifies the maximum acceptable body size.
	MaxAcceptableBody int64

	// NewClient is the MANDATORY factory to create a new client.
	NewClient func(model.Logger) model.HTTPClient

	// Out is the MANDATORY channel where we'll post results.
	Out chan ctrlHTTPResponse

	// URL is the MANDATORY URL to measure.
	URL string

	// Wg is MANDATORY and allows synchronizing with parent.
	Wg *sync.WaitGroup

	// searchForH3 is the OPTIONAL flag to decide whether to inspect Alt-Svc for HTTP/3 discovery
	searchForH3 bool
}

// httpDo performs the HTTP check.
func httpDo(ctx context.Context, config *httpConfig) {
	ol := measurexlite.NewOperationLogger(config.Logger, "GET %s", config.URL)
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer config.Wg.Done()
	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
	if err != nil {
		// fix: emit -1 like the old test helper does
		config.Out <- ctrlHTTPResponse{
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			Title:      "",
			Headers:    map[string]string{},
			StatusCode: -1,
		}
		ol.Stop(err)
		return
	}
	// The original test helper failed with extra headers while here
	// we're implementing (for now?) a more liberal approach.
	for k, vs := range config.Headers {
		switch strings.ToLower(k) {
		case "user-agent", "accept", "accept-language":
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	clnt := config.NewClient(config.Logger)
	defer clnt.CloseIdleConnections()
	resp, err := clnt.Do(req)
	if err != nil {
		// fix: emit -1 like the old test helper does
		config.Out <- ctrlHTTPResponse{
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			Title:      "",
			Headers:    map[string]string{},
			StatusCode: -1,
		}
		ol.Stop(err)
		return
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: config.MaxAcceptableBody}
	data, err := netxlite.ReadAllContext(ctx, reader)
	ol.Stop(err)

	h3Endpoint := ""
	if config.searchForH3 {
		h3Endpoint = discoverH3Endpoint(resp, req)
	}

	config.Out <- ctrlHTTPResponse{
		BodyLength:           int64(len(data)),
		DiscoveredH3Endpoint: h3Endpoint,
		Failure:              httpMapFailure(err),
		StatusCode:           int64(resp.StatusCode),
		Headers:              headers,
		Title:                measurexlite.WebGetTitle(string(data)),
	}
}

// Discovers an H3 endpoint by inspecting the Alt-Svc header in the first request-response pair
// of the redirect chain.
//
// TODO(kelmenhorst) Known limitations:
//   - This will not work for http:// URLs: Many/some/? hosts do not advertise h3 via Alt-Svc on a
//     cleartext HTTP response.
//     Thus, measuring http://cloudflare.com will not cause a h3 follow-up, but 
//     https://cloudflare.com will.
//   - We only consider the Alt-Svc binding of the very first request-response pair.
//     However, by using parseAltSvc we can later change the code to consider any request-response
//     pair without too much refactoring.
func discoverH3Endpoint(resp *http.Response, initReq *http.Request) string {
	firstResp, found := getFirstResponseInRedirectChain(resp)
	if !found {
		return ""
	}
	h3Endpoint := parseAltSvc(firstResp)
	if h3Endpoint == "" {
		return ""
	}
	// Examples:
	//
	//     Alt-Svc: h2="alt.example.com:443", h2=":443"
	//     Alt-Svc: h3-25=":443"; ma=3600, h2=":443"; ma=3600
	//
	// So here we need to handle both `alt.example.com:443` and `:443` cases.
	host, port, err := net.SplitHostPort(h3Endpoint)
	if err != nil {
		return ""
	}
	if host == "" {
		host = initReq.URL.Host
	}
	return net.JoinHostPort(host, port)
}

// search for the first HTTP response in the redirect chain
func getFirstResponseInRedirectChain(resp *http.Response) (*http.Response, bool) {
	// The default std lib behavior is to stop redirecting after 10 consecutive requests.
	// Defensively we stop searching after 11.
	for i := 0; i < 11; i++ {
		request := resp.Request
		runtimex.Assert(request != nil, "expected Request != nil")
		if request.Response == nil {
			return resp, true
		}
		resp = request.Response
	}
	return nil, false
}

func parseAltSvc(resp *http.Response) string { 
	altsvc := resp.Header.Get("Alt-Svc")
	// Syntax:
	//
	// Alt-Svc: clear
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>
	// Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>; persist=1
	//
	// Multiple entries may be separated by comma.
	//
	// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc
	entries := strings.Split(altsvc, ",")

	for _, entry := range entries {
		parts := strings.Split(entry, ";")
		runtimex.Assert(len(parts) > 0, "expected at least one entry in strings.Split result")

		_, alt_authority, _ := strings.Cut(parts[0], "h3=")
		if alt_authority == "" {
			continue
		}
		alt_authority = strings.TrimPrefix(alt_authority, "\"")
		alt_authority = strings.TrimSuffix(alt_authority, "\"")
		return alt_authority
	}
	return ""
}

// httpMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L361
func httpMapFailure(err error) *string {
	failure := newfailure(err)
	failedOperation := tracex.NewFailedOperation(err)
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureDNSNXDOMAINError,
			netxlite.FailureDNSNoAnswer,
			netxlite.FailureDNSNonRecoverableFailure,
			netxlite.FailureDNSRefusedError,
			netxlite.FailureDNSServerMisbehaving,
			netxlite.FailureDNSTemporaryFailure:
			// Strangely the HTTP code uses the more broad
			// dns_lookup_error and does not check for
			// the NXDOMAIN-equivalent-error dns_name_error
			s := "dns_lookup_error"
			return &s
		case netxlite.FailureGenericTimeoutError:
			// The old TH would return "dns_lookup_error" when
			// there is a timeout error during the DNS phase of HTTP.
			switch failedOperation {
			case nil:
				// nothing
			default:
				switch *failedOperation {
				case netxlite.ResolveOperation:
					s := "dns_lookup_error"
					return &s
				}
			}
			return failure // already using the same name
		case netxlite.FailureConnectionRefused:
			s := "connection_refused_error"
			return &s
		default:
			s := "unknown_error"
			return &s
		}
	}
}
