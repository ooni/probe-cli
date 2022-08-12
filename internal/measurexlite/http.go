package measurexlite

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// NewArchivalHTTPRequestResult creates a new model.ArchivalHTTPRequestResult.
//
// Arguments:
//
// - txp is the HTTP transport used for the HTTP transaction;
//
// - req is the certainly-non-nil HTTP request;
//
// - resp is the possibly-nil HTTP response;
//
// - maxRespBodySize is the maximum body snapshot size;
//
// - body is the possibly-nil HTTP response body;
//
// - err is the possibly-nil error that occurred during the HTTP transaction.
func (tx *Trace) NewArchivalHTTPRequestResult(
	txp model.HTTPTransport, req *http.Request, resp *http.Response, maxRespBodySize int64,
	body []byte, err error) *model.ArchivalHTTPRequestResult {
	ev := &model.ArchivalHTTPRequestResult{
		Failure: tracex.NewFailure(err),
		Request: model.ArchivalHTTPRequest{
			Body:            model.ArchivalMaybeBinaryData{},
			BodyIsTruncated: false,
			HeadersList:     newHTTPHeaderList(req.Header),
			Headers:         newHTTPHeaderMap(req.Header),
			Method:          req.Method,
			Tor:             model.ArchivalHTTPTor{},
			Transport:       txp.Network(),
			URL:             req.URL.String(),
		},
		Response: model.ArchivalHTTPResponse{
			Body:            model.ArchivalMaybeBinaryData{},
			BodyIsTruncated: false,
			Code:            0,
			HeadersList:     []model.ArchivalHTTPHeader{},
			Headers:         map[string]model.ArchivalMaybeBinaryData{},
			Locations:       []string{},
		},
		T:             tx.TimeSince(tx.ZeroTime).Seconds(),
		TransactionID: tx.Index,
	}
	if resp != nil {
		if body != nil {
			ev.Response.Body.Value = string(body)
			ev.Response.BodyIsTruncated = int64(len(body)) >= maxRespBodySize
		}
		ev.Response.Code = int64(resp.StatusCode)
		ev.Response.HeadersList = newHTTPHeaderList(resp.Header)
		ev.Response.Headers = newHTTPHeaderMap(resp.Header)
		loc, err := resp.Location()
		if err == nil {
			ev.Response.Locations = append(ev.Response.Locations, loc.String())
		}
	}
	return ev
}

// newHTTPHeaderList creates a list representation of HTTP headers
func newHTTPHeaderList(header http.Header) (out []model.ArchivalHTTPHeader) {
	for key, values := range header {
		for _, value := range values {
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
