package netplumbing

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// RoundTrip send an HTTP request and returns the response.
func (txp *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.roundTripWrapError(req)
}

// roundTripWrapError wraps the returned error using ErrHTTPRoundTrip
func (txp *Transport) roundTripWrapError(req *http.Request) (*http.Response, error) {
	resp, err := txp.roundTripEmitLogs(req)
	if err != nil {
		return nil, &ErrHTTPRoundTrip{err}
	}
	return resp, nil
}

// ErrHTTPRoundTrip wraps an error occurred during the HTTP round trip.
type ErrHTTPRoundTrip struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrHTTPRoundTrip) Unwrap() error {
	return err.error
}

// roundTripEmitLogs emits the round trip logs.
func (txp *Transport) roundTripEmitLogs(req *http.Request) (*http.Response, error) {
	log := txp.logger(req.Context())
	log.Debugf("> %s %s", req.Method, req.URL)
	for key, values := range req.Header {
		for _, value := range values {
			log.Debugf("> %s: %s", key, value)
		}
	}
	log.Debug(">")
	resp, err := txp.roundTripMaybeTrace(req)
	if err != nil {
		log.Debugf("< %s", err)
		return nil, err
	}
	log.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			log.Debugf("< %s: %s", key, value)
		}
	}
	log.Debug("<")
	return resp, nil
}

// roundTripMaybeTrace enables tracing if needed.
func (txp *Transport) roundTripMaybeTrace(req *http.Request) (*http.Response, error) {
	if th := ContextTraceHeader(req.Context()); th != nil {
		return txp.roundTripWithTraceHeader(req, th)
	}
	return txp.roundTripMaybeOverride(req)
}

// roundTripWithTraceHeader traces the round trip.
func (txp *Transport) roundTripWithTraceHeader(
	req *http.Request, th *TraceHeader) (*http.Response, error) {
	ev := &HTTPRoundTripTrace{
		Method:         req.Method,
		URL:            req.URL.String(),
		RequestHeaders: req.Header,
	}
	defer func() {
		ev.EndTime = time.Now()
		th.add(ev)
	}()
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			ev.Error = err
			return nil, err
		}
		ev.RequestBody = data
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	ev.StartTime = time.Now()
	resp, err := txp.roundTripMaybeOverride(req)
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.StatusCode = resp.StatusCode
	ev.RequestHeaders = resp.Header
	iocloser := resp.Body
	defer iocloser.Close() // close original body
	reader := io.LimitReader(resp.Body, int64(txp.maxBodySize()))
	data, err := ioutil.ReadAll(reader)
	if errors.Is(err, io.EOF) && resp.Close {
		err = nil // we expected to hit the EOF
	}
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.ResponseBody = data
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, nil
}

// HTTPRoundTripTrace is a measurement collected during the HTTP round trip.
type HTTPRoundTripTrace struct {
	// Method is the request method.
	Method string

	// URL is the request URL.
	URL string

	// RequestHeaders contains the request headers.
	RequestHeaders http.Header

	// RequestBody contains the request body. This body is never
	// truncated but you may wanna truncate it before uploading
	// the measurement to the OONI servers.
	RequestBody []byte

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done. The duration of the round trip
	// also includes the time spent reading the response.
	EndTime time.Time

	// StatusCode contains the status code.
	StatusCode int

	// ResponseHeaders contains the response headers.
	ResponseHeaders http.Header

	// ResponseBody contains the response body. This body is
	// truncated if larger than Tracer.MaxBodySize. You likely
	// want to further truncate it before uploading data to
	// the OONI servers to save bandwidth.
	ResponseBody []byte

	// Error contains the error.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *HTTPRoundTripTrace) Kind() string {
	return TraceKindHTTPRoundTrip
}

// maxBodySize returns the maximum allowed body size when tracing.
func (txp *Transport) maxBodySize() int {
	// TODO(bassosimone): make this configurable.
	return 1 << 24
}

// roundTripMaybeOverride uses either the default or the overriden round tripper.
func (txp *Transport) roundTripMaybeOverride(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	var t http.RoundTripper = txp.RoundTripper
	if config := ContextConfig(ctx); config != nil && config.HTTPTransport != nil {
		t = config.HTTPTransport
	}
	return t.RoundTrip(req)
}
