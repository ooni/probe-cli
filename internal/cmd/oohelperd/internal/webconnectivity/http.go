package webconnectivity

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/tracex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// CtrlHTTPResponse is the result of the HTTP check performed by
// the Web Connectivity test helper.
type CtrlHTTPResponse = webconnectivity.ControlHTTPRequestResult

// HTTPConfig configures the HTTP check.
type HTTPConfig struct {
	Client            *http.Client
	Headers           map[string][]string
	MaxAcceptableBody int64
	Out               chan CtrlHTTPResponse
	URL               string
	Wg                *sync.WaitGroup
}

// HTTPDo performs the HTTP check.
func HTTPDo(ctx context.Context, config *HTTPConfig) {
	defer config.Wg.Done()
	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
	if err != nil {
		config.Out <- CtrlHTTPResponse{ // fix: emit -1 like the old test helper does
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			StatusCode: -1,
			Headers:    map[string]string{},
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
	resp, err := config.Client.Do(req)
	if err != nil {
		config.Out <- CtrlHTTPResponse{ // fix: emit -1 like old test helper does
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			StatusCode: -1,
			Headers:    map[string]string{},
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
	config.Out <- CtrlHTTPResponse{
		BodyLength: int64(len(data)),
		Failure:    httpMapFailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
		Title:      webconnectivity.GetTitle(string(data)),
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
