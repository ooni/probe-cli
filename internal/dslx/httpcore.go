package dslx

//
// HTTP measurements core
//

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

// HTTPConnection is an HTTP connection bound to a TCP, TLS or QUIC connection
// that would use such a connection only and for any input URL. You generally
// use [HTTPTransportTCP], [HTTPTransportTLS] or [HTTPTransportQUIC] to
// create a new instance; if you want to initialize manually, make sure you
// init the fields marked as MANDATORY.
type HTTPConnection struct {
	// Address is the MANDATORY address we're connected to.
	Address string

	// Domain is the OPTIONAL domain from which the address was resolved.
	Domain string

	// Network is the MANDATORY network used by the underlying conn.
	Network string

	// Scheme is the MANDATORY URL scheme to use.
	Scheme string

	// TLSNegotiatedProtocol is the OPTIONAL negotiated protocol.
	TLSNegotiatedProtocol string

	// Trace is the MANDATORY trace we're using.
	Trace Trace

	// Transport is the MANDATORY HTTP transport we're using.
	Transport model.HTTPTransport
}

// HTTPRequestOption is an option you can pass to HTTPRequest.
type HTTPRequestOption func(req *http.Request)

// HTTPRequestOptionAccept sets the Accept header.
func HTTPRequestOptionAccept(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", value)
	}
}

// HTTPRequestOptionAcceptLanguage sets the Accept header.
func HTTPRequestOptionAcceptLanguage(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.Header.Set("Accept-Language", value)
	}
}

// HTTPRequestOptionHost sets the Host header.
func HTTPRequestOptionHost(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.URL.Host = value
		req.Host = value
	}
}

// HTTPRequestOptionHost sets the request method.
func HTTPRequestOptionMethod(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.Method = value
	}
}

// HTTPRequestOptionReferer sets the Referer header.
func HTTPRequestOptionReferer(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.Header.Set("Referer", value)
	}
}

// HTTPRequestOptionURLPath sets the URL path.
func HTTPRequestOptionURLPath(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.URL.Path = value
	}
}

// HTTPRequestOptionUserAgent sets the UserAgent header.
func HTTPRequestOptionUserAgent(value string) HTTPRequestOption {
	return func(req *http.Request) {
		req.Header.Set("User-Agent", value)
	}
}

// HTTPRequest issues an HTTP request using a transport and returns a response.
func HTTPRequest(rt Runtime, options ...HTTPRequestOption) Func[*HTTPConnection, *HTTPResponse] {
	return Operation[*HTTPConnection, *HTTPResponse](func(ctx context.Context, input *HTTPConnection) *Maybe[*HTTPResponse] {
		// setup
		const timeout = 10 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var (
			body         []byte
			observations []*Observations
			resp         *http.Response
		)

		// create HTTP request
		req, err := httpNewRequest(ctx, input, rt.Logger(), options...)
		if err == nil {

			// start the operation logger
			ol := logx.NewOperationLogger(
				rt.Logger(),
				"[#%d] HTTPRequest %s with %s/%s host=%s",
				input.Trace.Index(),
				req.URL.String(),
				input.Address,
				input.Network,
				req.Host,
			)

			// perform HTTP transaction and collect the related observations
			resp, body, observations, err = httpRoundTrip(ctx, input, req)

			// stop the operation logger
			ol.Stop(err)
		}

		// merge and save observations
		observations = append(observations, maybeTraceToObservations(input.Trace)...)
		rt.SaveObservations(observations...)

		state := &HTTPResponse{
			Address:                  input.Address,
			Domain:                   input.Domain,
			HTTPRequest:              req,  // possibly nil
			HTTPResponse:             resp, // possibly nil
			HTTPResponseBodySnapshot: body, // possibly nil
			Network:                  input.Network,
			Trace:                    input.Trace,
		}

		return &Maybe[*HTTPResponse]{
			Error: err,
			State: state,
		}
	})
}

// httpNewRequest is a convenience function for creating a new request.
func httpNewRequest(
	ctx context.Context, input *HTTPConnection, logger model.Logger, options ...HTTPRequestOption) (*http.Request, error) {
	// create the default HTTP request
	URL := &url.URL{
		Scheme:      input.Scheme,
		Opaque:      "",
		User:        nil,
		Host:        httpNewURLHost(input, logger),
		Path:        "/",
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Go would use URL.Host as "Host" header anyways in case we leave req.Host empty.
	// We already set it here so that we can use req.Host for logging.
	req.Host = URL.Host

	// apply the user-specified options
	for _, option := range options {
		option(req)
	}

	// req.Header["Host"] is ignored by Go but we want to have it in the measurement
	// to reflect what we think has been sent as HTTP headers.
	req.Header.Set("Host", req.Host)
	return req, nil
}

// httpNewURLHost computes the URL host to use.
func httpNewURLHost(input *HTTPConnection, logger model.Logger) string {
	if input.Domain != "" {
		return input.Domain
	}
	addr, port, err := net.SplitHostPort(input.Address)
	if err != nil {
		logger.Warnf("httpRequestFunc: cannot SplitHostPort for input.Address")
		return input.Address
	}
	switch {
	case port == "80" && input.Scheme == "http":
		return addr
	case port == "443" && input.Scheme == "https":
		return addr
	default:
		return input.Address // with port only if port is nonstandard
	}
}

// httpRoundTrip performs the actual HTTP round trip
func httpRoundTrip(
	ctx context.Context,
	input *HTTPConnection,
	req *http.Request,
) (*http.Response, []byte, []*Observations, error) {
	const maxbody = 1 << 19 // TODO(bassosimone): allow to configure this value?
	started := input.Trace.TimeSince(input.Trace.ZeroTime())

	// manually create a single 1-length observations structure because
	// the trace cannot automatically capture HTTP events
	observations := []*Observations{
		NewObservations(),
	}

	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			input.Trace.Index(),
			started,
			"http_transaction_start",
			input.Trace.Tags()...,
		))

	resp, err := input.Transport.RoundTrip(req)
	var body []byte
	if err == nil {
		defer resp.Body.Close()

		// create sampler for measuring throttling
		sampler := throttling.NewSampler(input.Trace)
		defer sampler.Close()

		// read a snapshot of the response body
		reader := io.LimitReader(resp.Body, maxbody)
		body, err = netxlite.ReadAllContext(ctx, reader) // TODO: enable streaming and measure speed

		// collect and save download speed samples
		samples := sampler.ExtractSamples()
		observations[0].NetworkEvents = append(observations[0].NetworkEvents, samples...)
	}
	finished := input.Trace.TimeSince(input.Trace.ZeroTime())

	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			input.Trace.Index(),
			finished,
			"http_transaction_done",
			input.Trace.Tags()...,
		))

	observations[0].Requests = append(observations[0].Requests,
		measurexlite.NewArchivalHTTPRequestResult(
			input.Trace.Index(),
			started,
			input.Network,
			input.Address,
			input.TLSNegotiatedProtocol,
			input.Transport.Network(),
			req,
			resp,
			maxbody,
			body,
			err,
			finished,
			input.Trace.Tags()...,
		))

	return resp, body, observations, err
}

// HTTPResponse is the response generated by an HTTP requests. Generally
// obtained by HTTPRequest().Apply. To init manually, init at least MANDATORY fields.
type HTTPResponse struct {
	// Address is the MANDATORY address we're connected to.
	Address string

	// Domain is the OPTIONAL domain from which we determined Address.
	Domain string

	// HTTPRequest is the possibly-nil HTTP request.
	HTTPRequest *http.Request

	// HTTPResponse is the HTTP response or nil if Err != nil.
	HTTPResponse *http.Response

	// HTTPResponseBodySnapshot is the response body or nil if Err != nil.
	HTTPResponseBodySnapshot []byte

	// Network is the MANDATORY network we're connected to.
	Network string

	// Trace is the MANDATORY trace we're using. The trace is drained
	// when you call the Observations method.
	Trace Trace
}
