package main

//
// HTTP measurements
//

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

	// MaxAcceptableBody is MANDATORY and specifies the maximum acceptable body size.
	MaxAcceptableBody int64

	// NewClient is the MANDATORY factory to create a new client.
	NewClient func() model.HTTPClient

	// Out is the MANDATORY channel where we'll post results.
	Out chan ctrlHTTPResponse

	// URL is the MANDATORY URL to measure.
	URL string

	// Wg is MANDATORY and allows synchronizing with parent.
	Wg *sync.WaitGroup
}

// httpDo performs the HTTP check.
func httpDo(ctx context.Context, config *httpConfig) {
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
	clnt := config.NewClient()
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
		return
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: config.MaxAcceptableBody}
	data, err := netxlite.ReadAllContext(ctx, reader)
	config.Out <- ctrlHTTPResponse{
		BodyLength: int64(len(data)),
		Failure:    httpMapFailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
		Title:      measurexlite.WebGetTitle(string(data)),
	}
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
