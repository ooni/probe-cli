package urlgetter

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPTransaction performs an HTTP transaction using the given URL.
func (rx *Runner) HTTPTransaction(ctx context.Context, config *Config, URL *url.URL) error {
	// TODO(bassosimone): honor the case where the user disabled HTTP redirection

	// TODO(bassosimone): we should also make sure we're correctly dealing with
	// errors and we always wrap the errors we return here

	for idx := 0; idx < 10; idx++ {
		// perform the round trip
		resp, err := rx.HTTPRoundTrip(ctx, config, URL)

		// handle the case of failure
		if err != nil {
			return err
		}

		// close the connection
		resp.Close()

		// handle the case of redirect
		if !httpRedirectIsRedirect(resp) {
			return nil
		}
		// TODO(bassosimone): not 100% convinced whether what follows
		// should be implemented as such or be different
		if err := httpValidateRedirect(resp); err != nil {
			return err
		}

		// TODO(bassosimone): this code is broken because it does not handle
		// cookies and we need cookies for some HTTP redirects to work

		// clone the original configuration
		config = config.Clone()

		// set the referer header to be the original URL
		config.HTTPReferer = URL.String()

		// replace the original URL with the location
		URL = resp.Location
	}

	// TODO(bassosimone): make sure the error we're using here is correct
	return errors.New("too many HTTP redirects")
}

// HTTPRoundTrip measures an HTTP or HTTPS URL.
func (rx *Runner) HTTPRoundTrip(ctx context.Context, config *Config, URL *url.URL) (*HTTPResponse, error) {
	switch URL.Scheme {
	case "http":
		return rx.HTTPRoundTripCleartext(ctx, config, URL)

	case "https":
		return rx.HTTPRoundTripSecure(ctx, config, URL)

	default:
		return nil, ErrUnknownURLScheme
	}
}

// HTTPRoundTripCleartext measures an HTTP URL.
func (rx *Runner) HTTPRoundTripCleartext(ctx context.Context, config *Config, URL *url.URL) (*HTTPResponse, error) {
	// establish a TCP connection
	conn, err := rx.tcpConnect(ctx, config, URL)
	if err != nil {
		return nil, err
	}

	// perform round trip
	return rx.HTTPRoundTripOp(ctx, conn.AsHTTPConn(rx.Session.Logger()))
}

// HTTPRoundTripSecure measures an HTTPS URL.
func (rx *Runner) HTTPRoundTripSecure(ctx context.Context, config *Config, URL *url.URL) (*HTTPResponse, error) {
	// establish a TLS connection
	conn, err := rx.tlsHandshake(ctx, config, URL)
	if err != nil {
		return nil, err
	}

	// perform round trip
	return rx.HTTPRoundTripOp(ctx, conn.AsHTTPConn(rx.Session.Logger()))
}

// HTTPResponse summarizes an HTTP response.
type HTTPResponse struct {
	// BodyReader allows to read the already read body snapshot
	// followed by zero or more bytes after the snapshot, depending
	// on whether the body is larger than the snapshot size.
	BodyReader io.Reader

	// Conn is the conn.
	Conn net.Conn

	// Location may contain the location if we're redirected.
	Location *url.URL

	// Status contains the status code.
	Status int
}

// Close implements io.Closer.
func (rx *HTTPResponse) Close() error {
	return rx.Conn.Close()
}

// HTTPConn is an established HTTP connection.
type HTTPConn struct {
	// Config is the original config.
	Config *Config

	// Conn is the conn.
	Conn net.Conn

	// Network is the network we're using.
	Network string

	// RemoteAddress is the remote address we're using.
	RemoteAddress string

	// TLSNegotiatedProtocol is the negotiated TLS protocol.
	TLSNegotiatedProtocol string

	// Trace is the trace.
	Trace *measurexlite.Trace

	// Transport is the single-use HTTP transport.
	Transport model.HTTPTransport

	// URL is the original URL.
	URL *url.URL
}

var _ io.Closer = &TLSConn{}

// Close implements io.Closer.
func (tx *HTTPConn) Close() error {
	return tx.Conn.Close()
}

func (cx *Config) method() (method string) {
	method = "GET"
	if cx.Method != "" {
		method = cx.Method
	}
	return
}

// HTTPRoundTripOp performs an HTTP round trip.
func (rx *Runner) HTTPRoundTripOp(ctx context.Context, input *HTTPConn) (*HTTPResponse, error) {
	// enforce timeout
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// create HTTP request
	req, err := rx.newHTTPRequest(ctx, input)
	if err != nil {
		// make sure we close the conn
		input.Close()
		return nil, err
	}

	// start operation logger
	ol := logx.NewOperationLogger(
		rx.Session.Logger(),
		"[#%d] %s %s with %s/%s host=%s",
		input.Trace.Index(),
		input.Config.method(),
		req.URL.String(),
		input.RemoteAddress,
		input.Network,
		req.Host,
	)

	// perform HTTP round trip
	resp, err := rx.httpRoundTripOp(ctx, input, req)

	// stop the operation logger
	ol.Stop(err)

	// handle failures
	if err != nil {
		// make sure we close the conn
		input.Close()

		// attempt to set top level failure and failed operation
		rx.TestKeys.MaybeSetFailedOperation(netxlite.HTTPRoundTripOperation)
		rx.TestKeys.MaybeSetFailure(err.Error())

		return nil, err
	}

	// otherwise return the response
	return resp, nil
}

func (rx *Runner) newHTTPRequest(ctx context.Context, conn *HTTPConn) (*http.Request, error) {
	// create the default HTTP request
	req, err := http.NewRequestWithContext(ctx, conn.Config.method(), conn.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Go would use URL.Host as "Host" header anyways in case we leave req.Host empty.
	// We already set it here so that we can use req.Host for logging.
	req.Host = conn.URL.Host

	// apply headers
	req.Header.Set("Accept", model.HTTPHeaderAccept)
	req.Header.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
	req.Header.Set("Referer", conn.Config.HTTPReferer)
	req.Header.Set("User-Agent", model.HTTPHeaderUserAgent)

	// req.Header["Host"] is ignored by Go but we want to have it in the measurement
	// to reflect what we think has been sent as HTTP headers.
	req.Header.Set("Host", req.Host)
	return req, nil
}

func (rx *Runner) httpRoundTripOp(ctx context.Context, conn *HTTPConn, req *http.Request) (*HTTPResponse, error) {
	// define the maximum body snapshot size
	const snapSize = 1 << 19

	// register when we started
	started := conn.Trace.TimeSince(conn.Trace.ZeroTime())

	// emit the beginning of the HTTP transaction
	rx.TestKeys.AppendNetworkEvents(measurexlite.NewAnnotationArchivalNetworkEvent(
		conn.Trace.Index(),
		started,
		"http_transaction_start",
		conn.Trace.Tags()...,
	))

	// perform the round trip
	resp, err := conn.Transport.RoundTrip(req)

	// on success also read a snapshot of the response body
	var body []byte
	if err == nil {
		// read a snapshot of the response body
		reader := io.LimitReader(resp.Body, snapSize)
		body, err = netxlite.StreamAllContext(ctx, reader)
	}

	// register when we finished
	finished := conn.Trace.TimeSince(conn.Trace.ZeroTime())

	// emit the end of the HTTP transaction
	rx.TestKeys.AppendNetworkEvents(measurexlite.NewAnnotationArchivalNetworkEvent(
		conn.Trace.Index(),
		started,
		"http_transaction_done",
		conn.Trace.Tags()...,
	))

	// emit the HTTP request event
	rx.TestKeys.PrependRequests(measurexlite.NewArchivalHTTPRequestResult(
		conn.Trace.Index(),
		started,
		conn.Network,
		conn.RemoteAddress,
		conn.TLSNegotiatedProtocol,
		conn.Transport.Network(),
		req,
		resp,
		snapSize,
		body,
		err,
		finished,
		conn.Trace.Tags()...,
	))

	// produce a response or an error
	return rx.httpFinish(conn.Conn, resp, err)
}

func (rx *Runner) httpFinish(conn net.Conn, resp *http.Response, err error) (*HTTPResponse, error) {
	// handle the case of failure first
	if err != nil {
		// make sure we do not leak the conn
		conn.Close()
		return nil, err
	}

	// get the location
	loc, _ := resp.Location()

	// fill and return the minimal HTTP response
	hresp := &HTTPResponse{
		BodyReader: resp.Body, // TODO(bassosimone): not consistent with docs: should use io.MultiReader
		Conn:       conn,
		Location:   loc,
		Status:     resp.StatusCode,
	}
	return hresp, nil
}
