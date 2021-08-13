package internal

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/iox"
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
		config.Out <- CtrlHTTPResponse{Failure: newfailure(err)}
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
		config.Out <- CtrlHTTPResponse{Failure: newfailure(err)}
		return
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: config.MaxAcceptableBody}
	data, err := iox.ReadAllContext(ctx, reader)
	config.Out <- CtrlHTTPResponse{
		BodyLength: int64(len(data)),
		Failure:    newfailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
		Title:      webconnectivity.GetTitle(string(data)),
	}
}
