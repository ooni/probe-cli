package tracex

//
// HTTP
//

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// SaverMetadataHTTPTransport is a RoundTripper that saves
// events related to HTTP request and response metadata
type SaverMetadataHTTPTransport struct {
	model.HTTPTransport
	Saver *Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverMetadataHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	txp.Saver.Write(Event{
		HTTPHeaders: httpCloneHeaders(req),
		HTTPMethod:  req.Method,
		HTTPURL:     req.URL.String(),
		Transport:   txp.HTTPTransport.Network(),
		Name:        "http_request_metadata",
		Time:        time.Now(),
	})
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	txp.Saver.Write(Event{
		HTTPHeaders:    resp.Header,
		HTTPStatusCode: resp.StatusCode,
		Name:           "http_response_metadata",
		Time:           time.Now(),
	})
	return resp, err
}

// httpCCloneHeaders returns a clone of the headers where we have
// also set the host header, which normally is not set by
// golang until it serializes the request itself.
func httpCloneHeaders(req *http.Request) http.Header {
	header := req.Header.Clone()
	if req.Host != "" {
		header.Set("Host", req.Host)
	} else {
		header.Set("Host", req.URL.Host)
	}
	return header
}

// SaverTransactionHTTPTransport is a RoundTripper that saves
// events related to the HTTP transaction
type SaverTransactionHTTPTransport struct {
	model.HTTPTransport
	Saver *Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverTransactionHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	txp.Saver.Write(Event{
		Name: "http_transaction_start",
		Time: time.Now(),
	})
	resp, err := txp.HTTPTransport.RoundTrip(req)
	txp.Saver.Write(Event{
		Err:  err,
		Name: "http_transaction_done",
		Time: time.Now(),
	})
	return resp, err
}

// SaverBodyHTTPTransport is a RoundTripper that saves
// body events occurring during the round trip
type SaverBodyHTTPTransport struct {
	model.HTTPTransport
	Saver        *Saver
	SnapshotSize int
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverBodyHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	const defaultSnapSize = 1 << 17
	snapsize := defaultSnapSize
	if txp.SnapshotSize != 0 {
		snapsize = txp.SnapshotSize
	}
	if req.Body != nil {
		data, err := httpSaverSnapRead(req.Context(), req.Body, snapsize)
		if err != nil {
			return nil, err
		}
		req.Body = httpSaverCompose(data, req.Body)
		txp.Saver.Write(Event{
			DataIsTruncated: len(data) >= snapsize,
			Data:            data,
			Name:            "http_request_body_snapshot",
			Time:            time.Now(),
		})
	}
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	data, err := httpSaverSnapRead(req.Context(), resp.Body, snapsize)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body = httpSaverCompose(data, resp.Body)
	txp.Saver.Write(Event{
		DataIsTruncated: len(data) >= snapsize,
		Data:            data,
		Name:            "http_response_body_snapshot",
		Time:            time.Now(),
	})
	return resp, nil
}

func httpSaverSnapRead(ctx context.Context, r io.ReadCloser, snapsize int) ([]byte, error) {
	return netxlite.ReadAllContext(ctx, io.LimitReader(r, int64(snapsize)))
}

func httpSaverCompose(data []byte, r io.ReadCloser) io.ReadCloser {
	return httpSaverReadCloser{Closer: r, Reader: io.MultiReader(bytes.NewReader(data), r)}
}

type httpSaverReadCloser struct {
	io.Closer
	io.Reader
}

var _ model.HTTPTransport = SaverMetadataHTTPTransport{}
var _ model.HTTPTransport = SaverBodyHTTPTransport{}
var _ model.HTTPTransport = SaverTransactionHTTPTransport{}
