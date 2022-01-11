package httptransport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// SaverPerformanceHTTPTransport is a RoundTripper that saves
// performance events occurring during the round trip
type SaverPerformanceHTTPTransport struct {
	model.HTTPTransport
	Saver *trace.Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverPerformanceHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tracep := httptrace.ContextClientTrace(req.Context())
	if tracep == nil {
		tracep = &httptrace.ClientTrace{
			WroteHeaders: func() {
				txp.Saver.Write(trace.Event{Name: "http_wrote_headers", Time: time.Now()})
			},
			WroteRequest: func(httptrace.WroteRequestInfo) {
				txp.Saver.Write(trace.Event{Name: "http_wrote_request", Time: time.Now()})
			},
			GotFirstResponseByte: func() {
				txp.Saver.Write(trace.Event{
					Name: "http_first_response_byte", Time: time.Now()})
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), tracep))
	}
	return txp.HTTPTransport.RoundTrip(req)
}

// SaverMetadataHTTPTransport is a RoundTripper that saves
// events related to HTTP request and response metadata
type SaverMetadataHTTPTransport struct {
	model.HTTPTransport
	Saver *trace.Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverMetadataHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	txp.Saver.Write(trace.Event{
		HTTPHeaders: txp.CloneHeaders(req),
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
	txp.Saver.Write(trace.Event{
		HTTPHeaders:    resp.Header,
		HTTPStatusCode: resp.StatusCode,
		Name:           "http_response_metadata",
		Time:           time.Now(),
	})
	return resp, err
}

// CloneHeaders returns a clone of the headers where we have
// also set the host header, which normally is not set by
// golang until it serializes the request itself.
func (txp SaverMetadataHTTPTransport) CloneHeaders(req *http.Request) http.Header {
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
	Saver *trace.Saver
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp SaverTransactionHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	txp.Saver.Write(trace.Event{
		Name: "http_transaction_start",
		Time: time.Now(),
	})
	resp, err := txp.HTTPTransport.RoundTrip(req)
	txp.Saver.Write(trace.Event{
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
	Saver        *trace.Saver
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
		data, err := saverSnapRead(req.Context(), req.Body, snapsize)
		if err != nil {
			return nil, err
		}
		req.Body = saverCompose(data, req.Body)
		txp.Saver.Write(trace.Event{
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
	data, err := saverSnapRead(req.Context(), resp.Body, snapsize)
	err = ignoreExpectedEOF(err, resp)
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body = saverCompose(data, resp.Body)
	txp.Saver.Write(trace.Event{
		DataIsTruncated: len(data) >= snapsize,
		Data:            data,
		Name:            "http_response_body_snapshot",
		Time:            time.Now(),
	})
	return resp, nil
}

// ignoreExpectedEOF converts an error signalling the end of the body
// into a success. We know that we are in such condition when the
// resp.Close hint flag is set to true. (Thanks, stdlib!)
//
// See https://github.com/ooni/probe-engine/issues/1191 for an analysis
// of how this error was impacting measurements and data quality.
func ignoreExpectedEOF(err error, resp *http.Response) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) && resp.Close {
		return nil
	}
	return err
}

func saverSnapRead(ctx context.Context, r io.ReadCloser, snapsize int) ([]byte, error) {
	return netxlite.ReadAllContext(ctx, io.LimitReader(r, int64(snapsize)))
}

func saverCompose(data []byte, r io.ReadCloser) io.ReadCloser {
	return saverReadCloser{Closer: r, Reader: io.MultiReader(bytes.NewReader(data), r)}
}

type saverReadCloser struct {
	io.Closer
	io.Reader
}

var _ model.HTTPTransport = SaverPerformanceHTTPTransport{}
var _ model.HTTPTransport = SaverMetadataHTTPTransport{}
var _ model.HTTPTransport = SaverBodyHTTPTransport{}
var _ model.HTTPTransport = SaverTransactionHTTPTransport{}
