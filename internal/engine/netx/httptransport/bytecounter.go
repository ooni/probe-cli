package httptransport

import (
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
)

// ByteCountingTransport is a RoundTripper that counts bytes.
type ByteCountingTransport struct {
	RoundTripper
	Counter *bytecounter.Counter
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp ByteCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		req.Body = byteCountingBody{
			ReadCloser: req.Body, Account: txp.Counter.CountBytesSent}
	}
	txp.estimateRequestMetadata(req)
	resp, err := txp.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	txp.estimateResponseMetadata(resp)
	resp.Body = byteCountingBody{
		ReadCloser: resp.Body, Account: txp.Counter.CountBytesReceived}
	return resp, nil
}

func (txp ByteCountingTransport) estimateRequestMetadata(req *http.Request) {
	txp.Counter.CountBytesSent(len(req.Method))
	txp.Counter.CountBytesSent(len(req.URL.String()))
	for key, values := range req.Header {
		for _, value := range values {
			txp.Counter.CountBytesSent(len(key))
			txp.Counter.CountBytesSent(len(": "))
			txp.Counter.CountBytesSent(len(value))
			txp.Counter.CountBytesSent(len("\r\n"))
		}
	}
	txp.Counter.CountBytesSent(len("\r\n"))
}

func (txp ByteCountingTransport) estimateResponseMetadata(resp *http.Response) {
	txp.Counter.CountBytesReceived(len(resp.Status))
	for key, values := range resp.Header {
		for _, value := range values {
			txp.Counter.CountBytesReceived(len(key))
			txp.Counter.CountBytesReceived(len(": "))
			txp.Counter.CountBytesReceived(len(value))
			txp.Counter.CountBytesReceived(len("\r\n"))
		}
	}
	txp.Counter.CountBytesReceived(len("\r\n"))
}

type byteCountingBody struct {
	io.ReadCloser
	Account func(int)
}

func (r byteCountingBody) Read(p []byte) (int, error) {
	count, err := r.ReadCloser.Read(p)
	if count > 0 {
		r.Account(count)
	}
	return count, err
}

var _ RoundTripper = ByteCountingTransport{}
