package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

type (
	// CtrlResponse is the type of response returned by the test helper.
	CtrlResponse = webconnectivity.ControlResponse

	// ctrlRequest is the type of the request sent to the test helper.
	ctrlRequest = webconnectivity.ControlRequest
)

// The following errors may be returned by this implementation.
var (
	ErrHTTPStatusCode          = errors.New("oohelper: http status code indicates failure")
	ErrUnsupportedURLScheme    = errors.New("oohelper: unsupported URL scheme")
	ErrUnsupportedExplicitPort = errors.New("oohelper: unsupported explicit port")
	ErrEmptyURL                = errors.New("oohelper: empty server and/or target URL")
	ErrInvalidURL              = errors.New("oohelper: cannot parse URL")
	ErrCannotCreateRequest     = errors.New("oohelper: cannot create HTTP request")
	ErrCannotParseJSONReply    = errors.New("oohelper: cannot parse JSON reply")
)

// OOClient is a client for the OONI Web Connectivity test helper.
type OOClient struct {
	// HTTPClient is the HTTP client to use.
	HTTPClient *http.Client

	// Resolver is the resolver to user.
	Resolver netx.Resolver
}

// OOConfig contains configuration for the client.
type OOConfig struct {
	// ServerURL is the URL of the test helper server.
	ServerURL string

	// TargetURL is the URL that we want to measure.
	TargetURL string
}

// MakeTCPEndpoints constructs the list of TCP endpoints to send
// to the Web Connectivity test helper.
func MakeTCPEndpoints(URL *url.URL, addrs []string) ([]string, error) {
	var (
		port string
		out  []string
	)
	if URL.Host != URL.Hostname() {
		return nil, ErrUnsupportedExplicitPort
	}
	switch URL.Scheme {
	case "https":
		port = "443"
	case "http":
		port = "80"
	default:
		return nil, ErrUnsupportedURLScheme
	}
	for _, addr := range addrs {
		out = append(out, net.JoinHostPort(addr, port))
	}
	return out, nil
}

// Do sends a measurement request to the Web Connectivity test
// helper and receives the corresponding response.
func (oo OOClient) Do(ctx context.Context, config OOConfig) (*CtrlResponse, error) {
	if config.TargetURL == "" || config.ServerURL == "" {
		return nil, ErrEmptyURL
	}
	targetURL, err := url.Parse(config.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidURL, err.Error())
	}
	addrs, err := oo.Resolver.LookupHost(ctx, targetURL.Hostname())
	endpoints := []string{}
	if err == nil {
		endpoints, err = MakeTCPEndpoints(targetURL, addrs)
		if err != nil {
			return nil, err
		}
	}
	creq := ctrlRequest{
		HTTPRequest: config.TargetURL,
		HTTPRequestHeaders: map[string][]string{
			"Accept":          {httpheader.Accept()},
			"Accept-Language": {httpheader.AcceptLanguage()},
			"User-Agent":      {httpheader.UserAgent()},
		},
		TCPConnect: endpoints,
	}
	data, err := json.Marshal(creq)
	runtimex.PanicOnError(err, "oohelper: cannot marshal control request")
	log.Debugf("out: %s", string(data))
	req, err := http.NewRequestWithContext(ctx, "POST", config.ServerURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotCreateRequest, err.Error())
	}
	req.Header.Add("user-agent", fmt.Sprintf(
		"oohelper/%s ooniprobe-engine/%s", version.Version, version.Version,
	))
	req.Header.Add("content-type", "application/json")
	resp, err := oo.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, ErrHTTPStatusCode
	}
	data, err = iox.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return nil, err
	}
	var cresp CtrlResponse
	if err := json.Unmarshal(data, &cresp); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCannotParseJSONReply, err.Error())
	}
	return &cresp, nil
}
