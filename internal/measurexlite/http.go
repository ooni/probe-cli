package measurexlite

//
// Support for generating HTTP traces
//

import (
	"net/http"
	"sort"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// NewArchivalHTTPRequestResult creates a new model.ArchivalHTTPRequestResult.
//
// Arguments:
//
// - index is the index of the trace;
//
// - started is when we started sending the request;
//
// - network is the underlying network in use ("tcp" or "udp");
//
// - address is the remote endpoint's address;
//
// - alpn is the negotiated ALPN or an empty string when not applicable;
//
// - transport is the HTTP transport's protocol we're using ("quic" or "tcp"): this field
// was introduced a long time ago to support QUIC measurements and we keep it for backwards
// compatibility but network, address, and alpn are much more informative;
//
// - req is the certainly-non-nil HTTP request;
//
// - resp is the possibly-nil HTTP response;
//
// - maxRespBodySize is the maximum body snapshot size;
//
// - body is the possibly-nil HTTP response body;
//
// - err is the possibly-nil error that occurred during the transaction;
//
// - finished is when we finished reading the response's body.
func NewArchivalHTTPRequestResult(index int64, started time.Duration, network, address, alpn string,
	transport string, req *http.Request, resp *http.Response, maxRespBodySize int64, body []byte, err error,
	finished time.Duration) *model.ArchivalHTTPRequestResult {
	return &model.ArchivalHTTPRequestResult{
		Network: network,
		Address: address,
		ALPN:    alpn,
		Failure: tracex.NewFailure(err),
		Request: model.ArchivalHTTPRequest{
			Body:            model.ArchivalMaybeBinaryData{},
			BodyIsTruncated: false,
			HeadersList:     newHTTPRequestHeaderList(req),
			Headers:         newHTTPRequestHeaderMap(req),
			Method:          httpRequestMethod(req),
			Tor:             model.ArchivalHTTPTor{},
			Transport:       transport, // kept for backward compat
			URL:             httpRequestURL(req),
		},
		Response: model.ArchivalHTTPResponse{
			Body:            httpResponseBody(body),
			BodyIsTruncated: httpResponseBodyIsTruncated(body, maxRespBodySize),
			Code:            httpResponseStatusCode(resp),
			HeadersList:     newHTTPResponseHeaderList(resp),
			Headers:         newHTTPResponseHeaderMap(resp),
			Locations:       httpResponseLocations(resp),
		},
		T0:            started.Seconds(),
		T:             finished.Seconds(),
		TransactionID: index,
	}
}

// httpRequestMethod returns the HTTP request method or an empty string
func httpRequestMethod(req *http.Request) (out string) {
	if req != nil {
		out = req.Method
	}
	return
}

// newHTTPRequestHeaderList calls newHTTPHeaderList with the request headers or
// return an empty array in case the request is nil.
func newHTTPRequestHeaderList(req *http.Request) []model.ArchivalHTTPHeader {
	m := http.Header{}
	if req != nil {
		m = req.Header
	}
	return newHTTPHeaderList(m)
}

// newHTTPRequestHeaderMap calls newHTTPHeaderMap with the request headers or
// return an empty map in case the request is nil.
func newHTTPRequestHeaderMap(req *http.Request) map[string]model.ArchivalMaybeBinaryData {
	m := http.Header{}
	if req != nil {
		m = req.Header
	}
	return newHTTPHeaderMap(m)
}

// httpRequestURL returns the req.URL.String() or an empty string.
func httpRequestURL(req *http.Request) (out string) {
	if req != nil && req.URL != nil {
		out = req.URL.String()
	}
	return
}

// httpResponseBody returns the response body, if possible, or an empty body.
func httpResponseBody(body []byte) (out model.ArchivalMaybeBinaryData) {
	if body != nil {
		out.Value = string(body)
	}
	return
}

// httpResponseBodyIsTruncated determines whether the body is truncated (if possible)
func httpResponseBodyIsTruncated(body []byte, maxSnapSize int64) (out bool) {
	if len(body) > 0 && maxSnapSize > 0 {
		out = int64(len(body)) >= maxSnapSize
	}
	return
}

// httpResponseStatusCode returns the status code, if possible
func httpResponseStatusCode(resp *http.Response) (code int64) {
	if resp != nil {
		code = int64(resp.StatusCode)
	}
	return
}

// newHTTPResponseHeaderList calls newHTTPHeaderList with the request headers or
// return an empty array in case the request is nil.
func newHTTPResponseHeaderList(resp *http.Response) (out []model.ArchivalHTTPHeader) {
	m := http.Header{}
	if resp != nil {
		m = resp.Header
	}
	return newHTTPHeaderList(m)
}

// newHTTPResponseHeaderMap calls newHTTPHeaderMap with the request headers or
// return an empty map in case the request is nil.
func newHTTPResponseHeaderMap(resp *http.Response) (out map[string]model.ArchivalMaybeBinaryData) {
	m := http.Header{}
	if resp != nil {
		m = resp.Header
	}
	return newHTTPHeaderMap(m)
}

// httpResponseLocations returns the locations inside the response (if possible)
func httpResponseLocations(resp *http.Response) []string {
	if resp == nil {
		return []string{}
	}
	loc, err := resp.Location()
	if err != nil {
		return []string{}
	}
	return []string{loc.String()}
}

// newHTTPHeaderList creates a list representation of HTTP headers
func newHTTPHeaderList(header http.Header) (out []model.ArchivalHTTPHeader) {
	out = []model.ArchivalHTTPHeader{}
	keys := []string{}
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys) // ensure the output is consistent, which helps with testing
	for _, key := range keys {
		for _, value := range header[key] {
			out = append(out, model.ArchivalHTTPHeader{
				Key: key,
				Value: model.ArchivalMaybeBinaryData{
					Value: value,
				},
			})
		}
	}
	return
}

// newHTTPHeaderMap creates a map representation of HTTP headers
func newHTTPHeaderMap(header http.Header) (out map[string]model.ArchivalMaybeBinaryData) {
	out = make(map[string]model.ArchivalMaybeBinaryData)
	for key, values := range header {
		for _, value := range values {
			out[key] = model.ArchivalMaybeBinaryData{
				Value: value,
			}
			break
		}
	}
	return
}
