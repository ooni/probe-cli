package measurex

//
// WCTH (Web Connectivity Test Helper)
//
// We use the WCTH as an alternative DNS for gathering
// additional IP addresses to test, which is useful when
// your local DNS is censored.
//
// This code is merely here to bootstrap websteps and
// should be removed when we have a proper test helper.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// WCTHWorker is the Web Connectivity test helper worker.
type WCTHWorker struct {
	clnt   HTTPClient
	db     EventDB
	logger Logger
	mid    int64
	url    string
}

// NewWCTHWorker creates a new TestHelper instance using the
// web connectivity test helper protocol.
//
// Arguments:
//
// - measurementID is the measurement ID;
//
// - logger is the logger to use;
//
// - db is the database to use;
//
// - clnt is the HTTP client to use;
//
// - URL is the WCTH service URL.
//
// All arguments are mandatory.
func NewWCTHWorker(measurementID int64,
	logger Logger, db EventDB, clnt HTTPClient, URL string) *WCTHWorker {
	return &WCTHWorker{
		db:     db,
		logger: logger,
		clnt:   clnt,
		url:    URL,
		mid:    measurementID,
	}
}

var errWCTHRequestFailed = errors.New("wcth: request failed")

// Run runs the WCTH for the given URL and endpoints and creates
// measurements into the DB that derive on the WCTH response.
//
// CAVEAT: this implementation is very inefficient because the
// WCTH will fetch the whole redirection chain for every request
// but the WCTH is already there and it can bootstrap us.
func (w *WCTHWorker) Run(
	ctx context.Context, URL *url.URL, endpoints []string) (*WCTHResponse, error) {
	req, err := w.newHTTPRequest(ctx, URL, endpoints)
	if err != nil {
		return nil, err
	}
	resp, err := w.do(req)
	if err != nil {
		return nil, err
	}
	w.parseResp(URL, resp)
	return resp, nil
}

func (w *WCTHWorker) parseResp(URL *url.URL, resp *WCTHResponse) {
	w.db.InsertIntoLookupHost(&LookupHostEvent{
		Origin:        OriginTH,
		MeasurementID: w.mid,
		Network:       "system",
		Address:       "",
		Domain:        URL.Hostname(),
		Started:       0,
		Finished:      0,
		Error:         w.newError(resp.DNS.Failure),
		Addrs:         w.filterDNSAddrs(resp.DNS.Addrs),
	})
	for addr, status := range resp.TCPConnect {
		w.db.InsertIntoDial(&NetworkEvent{
			Origin:        OriginTH,
			MeasurementID: w.mid,
			ConnID:        0,
			Operation:     "connect",
			Network:       "tcp",
			RemoteAddr:    addr,
			LocalAddr:     "",
			Started:       0,
			Finished:      0,
			Error:         w.newError(status.Failure),
			Count:         0,
		})
	}
}

func (w *WCTHWorker) newHTTPRequest(ctx context.Context,
	URL *url.URL, endpoints []string) (*http.Request, error) {
	wtchReq := &wcthRequest{
		HTTPRequest:        URL.String(),
		HTTPRequestHeaders: NewHTTPRequestHeaderForMeasuring(),
		TCPConnect:         endpoints,
	}
	reqBody, err := json.Marshal(wtchReq)
	runtimex.PanicOnError(err, "json.Marshal failed")
	req, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("miniooni/%s", version.Version))
	return req, nil
}

func (w *WCTHWorker) do(req *http.Request) (*WCTHResponse, error) {
	resp, err := w.clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errWCTHRequestFailed
	}
	const maxResponseBodySize = 1 << 20 // limit test helper response size
	r := io.LimitReader(resp.Body, maxResponseBodySize)
	respBody, err := iox.ReadAllContext(req.Context(), r)
	if err != nil {
		return nil, err
	}
	var wcthResp WCTHResponse
	if err := json.Unmarshal(respBody, &wcthResp); err != nil {
		return nil, err
	}
	return &wcthResp, nil
}

func (w *WCTHWorker) filterDNSAddrs(addrs []string) (out []string) {
	for _, addr := range addrs {
		if net.ParseIP(addr) == nil {
			continue // WCTH also returns the CNAME
		}
		out = append(out, addr)
	}
	return
}

func (w *WCTHWorker) newError(failure *string) error {
	if failure != nil {
		return errors.New(*failure)
	}
	return nil
}

type wcthRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`
}

// WCTHTCPConnectResult contains the TCP connect result.
type WCTHTCPConnectResult struct {
	Status  bool    `json:"status"`
	Failure *string `json:"failure"`
}

// WCTHHTTPRequestResult contains the HTTP result.
type WCTHHTTPRequestResult struct {
	BodyLength int64             `json:"body_length"`
	Failure    *string           `json:"failure"`
	Title      string            `json:"title"`
	Headers    map[string]string `json:"headers"`
	StatusCode int64             `json:"status_code"`
}

// WCTHDNSResult contains the DNS result.
type WCTHDNSResult struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
}

// WCTHResponse is the response from the WCTH service.
type WCTHResponse struct {
	TCPConnect  map[string]WCTHTCPConnectResult `json:"tcp_connect"`
	HTTPRequest WCTHHTTPRequestResult           `json:"http_request"`
	DNS         WCTHDNSResult                   `json:"dns"`
}
