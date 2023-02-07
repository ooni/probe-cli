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
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPTransport is an HTTP transport bound to a TCP or TLS connection
// that would use such a connection only and for any input URL. You generally
// use HTTPTransportTCP or HTTPTransportTLS to create a new instance; if you
// want to initialize manually, make sure you init the MANDATORY fields.
type HTTPTransport struct {
	// Address is the MANDATORY address we're connected to.
	Address string

	// Domain is the OPTIONAL domain from which the address was resolved.
	Domain string

	// IDGenerator is the MANDATORY ID generator.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network used by the underlying conn.
	Network string

	// Scheme is the MANDATORY URL scheme to use.
	Scheme string

	// TLSNegotiatedProtocol is the OPTIONAL negotiated protocol.
	TLSNegotiatedProtocol string

	// Trace is the MANDATORY trace we're using.
	Trace *measurexlite.Trace

	// Transport is the MANDATORY HTTP transport we're using.
	Transport model.HTTPTransport

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}

// HTTPRequestOption is an option you can pass to HTTPRequest.
type HTTPRequestOption func(*httpRequestFunc)

// HTTPRequestOptionAccept sets the Accept header.
func HTTPRequestOptionAccept(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.Accept = value
	}
}

// HTTPRequestOptionAcceptLanguage sets the Accept header.
func HTTPRequestOptionAcceptLanguage(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.AcceptLanguage = value
	}
}

// HTTPRequestOptionHost sets the Host header.
func HTTPRequestOptionHost(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.Host = value
	}
}

// HTTPRequestOptionHost sets the request method.
func HTTPRequestOptionMethod(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.Method = value
	}
}

// HTTPRequestOptionReferer sets the Referer header.
func HTTPRequestOptionReferer(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.Referer = value
	}
}

// HTTPRequestOptionURLPath sets the URL path.
func HTTPRequestOptionURLPath(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.URLPath = value
	}
}

// HTTPRequestOptionUserAgent sets the UserAgent header.
func HTTPRequestOptionUserAgent(value string) HTTPRequestOption {
	return func(hrf *httpRequestFunc) {
		hrf.UserAgent = value
	}
}

// HTTPRequest issues an HTTP request using a transport and returns a response.
func HTTPRequest(options ...HTTPRequestOption) Func[*HTTPTransport, *Maybe[*HTTPResponse]] {
	f := &httpRequestFunc{}
	for _, option := range options {
		option(f)
	}
	return f
}

// httpRequestFunc is the Func returned by HTTPRequest.
type httpRequestFunc struct {
	// Accept is the OPTIONAL accept header.
	Accept string

	// AcceptLanguage is the OPTIONAL accept-language header.
	AcceptLanguage string

	// Host is the OPTIONAL host header.
	Host string

	// Method is the OPTIONAL method.
	Method string

	// Referer is the OPTIONAL referer header.
	Referer string

	// URLPath is the OPTIONAL URL path.
	URLPath string

	// UserAgent is the OPTIONAL user-agent header.
	UserAgent string
}

// Apply implements Func.
func (f *httpRequestFunc) Apply(
	ctx context.Context, input *HTTPTransport) *Maybe[*HTTPResponse] {
	// create HTTP request
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var (
		body         []byte
		observations []*Observations
		resp         *http.Response
	)

	req, err := f.newHTTPRequest(ctx, input)
	if err == nil {

		// start the operation logger
		ol := measurexlite.NewOperationLogger(
			input.Logger,
			"[#%d] HTTPRequest %s with %s/%s host=%s",
			input.Trace.Index,
			req.URL.String(),
			input.Address,
			input.Network,
			req.Host,
		)

		// perform HTTP transaction and collect the related observations
		resp, body, observations, err = f.do(ctx, input, req)

		// stop the operation logger
		ol.Stop(err)
	}

	observations = append(observations, maybeTraceToObservations(input.Trace)...)

	state := &HTTPResponse{
		Address:                  input.Address,
		Domain:                   input.Domain,
		HTTPRequest:              req,  // possibly nil
		HTTPResponse:             resp, // possibly nil
		HTTPResponseBodySnapshot: body, // possibly nil
		IDGenerator:              input.IDGenerator,
		Logger:                   input.Logger,
		Network:                  input.Network,
		Trace:                    input.Trace,
		ZeroTime:                 input.ZeroTime,
	}

	return &Maybe[*HTTPResponse]{
		Error:        err,
		Observations: observations,
		Skipped:      false,
		State:        state,
	}
}

func (f *httpRequestFunc) newHTTPRequest(
	ctx context.Context, input *HTTPTransport) (*http.Request, error) {
	URL := &url.URL{
		Scheme:      input.Scheme,
		Opaque:      "",
		User:        nil,
		Host:        f.urlHost(input),
		Path:        f.urlPath(),
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}

	method := "GET"
	if f.Method != "" {
		method = f.Method
	}

	req, err := http.NewRequestWithContext(ctx, method, URL.String(), nil)
	if err != nil {
		return nil, err
	}

	if v := f.Host; v != "" {
		req.Host = v
	} else {
		// Go would use URL.Host as "Host" header anyways in case we leave req.Host empty.
		// We already set it here so that we can use req.Host for logging.
		req.Host = URL.Host
	}
	// req.Header["Host"] is ignored by Go but we want to have it in the measurement
	// to reflect what we think has been sent as HTTP headers.
	req.Header.Set("Host", req.Host)

	if v := f.Accept; v != "" {
		req.Header.Set("Accept", v)
	}

	if v := f.AcceptLanguage; v != "" {
		req.Header.Set("Accept-Language", v)
	}

	if v := f.Referer; v != "" {
		req.Header.Set("Referer", v)
	}

	if v := f.UserAgent; v != "" { // not setting means using Go's default
		req.Header.Set("User-Agent", v)
	}

	return req, nil
}

func (f *httpRequestFunc) urlHost(input *HTTPTransport) string {
	if input.Domain != "" {
		return input.Domain
	}
	addr, port, err := net.SplitHostPort(input.Address)
	if err != nil {
		input.Logger.Warnf("httpRequestFunc: cannot SplitHostPort for input.Address")
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

func (f *httpRequestFunc) urlPath() string {
	if f.URLPath != "" {
		return f.URLPath
	}
	return "/"
}

func (f *httpRequestFunc) do(
	ctx context.Context,
	input *HTTPTransport,
	req *http.Request,
) (*http.Response, []byte, []*Observations, error) {
	const maxbody = 1 << 19 // TODO(bassosimone): allow to configure this value?
	started := input.Trace.TimeSince(input.Trace.ZeroTime)
	observations := []*Observations{{}} // one entry!

	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			input.Trace.Index,
			started,
			"http_transaction_start",
		))

	resp, err := input.Transport.RoundTrip(req)
	var body []byte
	if err == nil {
		defer resp.Body.Close()
		reader := io.LimitReader(resp.Body, maxbody)
		body, err = netxlite.ReadAllContext(ctx, reader) // TODO: enable streaming and measure speed
	}
	finished := input.Trace.TimeSince(input.Trace.ZeroTime)

	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			input.Trace.Index,
			finished,
			"http_transaction_done",
		))

	observations[0].Requests = append(observations[0].Requests,
		measurexlite.NewArchivalHTTPRequestResult(
			input.Trace.Index,
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

	// IDGenerator is the MANDATORY ID generator.
	IDGenerator *atomic.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// Network is the MANDATORY network we're connected to.
	Network string

	// Trace is the MANDATORY trace we're using. The trace is drained
	// when you call the Observations method.
	Trace *measurexlite.Trace

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time
}
