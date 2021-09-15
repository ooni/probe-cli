package measure

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/publicsuffix"
)

// NewCookieJar generates an http.CookieJar that implements
// proper filtering functionality to prevent domains from setting
// cookies for other, unrelated domains.
func NewCookieJar() http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	// rationale: as of 2021-09-15, the code always returns nil
	runtimex.PanicOnError(err, "cookiejar.New failed")
	return jar
}

// NewRequestHeadersForWebConnectivity returns the request
// headers we should always set for measuring according
// to the Web Connectivity specification.
func NewRequestHeadersForWebConnectivity() http.Header {
	header := make(http.Header)
	header.Set("Accept", httpheader.Accept())
	header.Set("Accept-Language", httpheader.AcceptLanguage())
	header.Set("User-Agent", httpheader.UserAgent())
	return header
}

// MaxBodySizeForScheme returns the maximum amount of
// bytes we will read from an HTTP response body before
// giving up and truncating the body. The amount of
// bytes may depend on the URL scheme, so this function
// takes as parameter the URL.
func MaxBodySizeForScheme(URL *url.URL) int64 {
	// See https://github.com/ooni/probe/issues/1727
	switch URL.Scheme {
	case "https":
		return 1 << 13
	default:
		return 1 << 17
	}
}

// NewHTTPRequestWithHostOverride creates an HTTPRequest
// data structure where we use a given host override along
// with the provided URL and cookies.
func NewHTTPRequestWithHostOverride(URL *url.URL,
	cookies http.CookieJar, hostHeader string) *HTTPRequest {
	return &HTTPRequest{
		URL:         URL.String(),
		Host:        hostHeader,
		Headers:     NewRequestHeadersForWebConnectivity(),
		Cookies:     cookies,
		MaxBodySize: MaxBodySizeForScheme(URL),
	}
}

// NewHTTPRequest creates a new HTTPRequest data
// structure using the given URL and cookies.
func NewHTTPRequest(URL *url.URL, cookies http.CookieJar) *HTTPRequest {
	return NewHTTPRequestWithHostOverride(URL, cookies, "")
}

// httpTransport allows sending requests and receiving responses.
type httpTransport = netxlite.HTTPTransport

// newHTTPTransportWithTCPConn creates a new HTTPTransport
// using the given TCP conn as its unique connection.
func newHTTPTransportWithTCPConn(logger Logger, conn net.Conn) httpTransport {
	d := netxlite.NewSingleUseDialer(conn)
	dt := netxlite.NewNullTLSDialer()
	return netxlite.NewHTTPTransport(logger, d, dt)
}

// newHTTPTransportWithTLSConn creates a new HTTPTransport
// using the given TLS conn as its unique connection.
func newHTTPTransportWithTLSConn(logger Logger, conn TLSConn) httpTransport {
	d := netxlite.NewNullDialer()
	dt := netxlite.NewSingleUseTLSDialer(conn)
	return netxlite.NewHTTPTransport(logger, d, dt)
}

// newHTTPTransportWithQUICSess creates a new HTTPTransport
// using the given QUIC sess as its unique connection.
func newHTTPTransportWithQUICSess(logger Logger, sess quic.EarlySession) httpTransport {
	d := netxlite.NewSingleUseQUICDialer(sess)
	return netxlite.NewHTTP3Transport(logger, d, &tls.Config{})
}

// HTTPRequestResponse contains the HTTP request and response.
type HTTPRequestResponse struct {
	// Request is the original request.
	Request *HTTPRequest `json:"request"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Failure is the error that occurred.
	Failure error `json:"failure"`

	// Response is the response. This field is nil
	// when there is a failure before we can at least
	// finish reading the response headers.
	Response *HTTPResponse `json:"response"`
}

// HTTPRequest is a request to get a resource.
//
// Make sure you fill all the fields marked as MANDATORY.
type HTTPRequest struct {
	// URL is the MANDATORY already-parsed target URL.
	URL string `json:"url"`

	// Host is the OPTIONAL host header to use.
	Host string `json:"host"`

	// Headers contains MANDATORY request headers.
	Headers http.Header `json:"headers"`

	// Cookies contains MANDATORY already-set cookies.
	Cookies http.CookieJar `json:"-"`

	// MaxBodySize contains the OPTIONAL maximum body size. If this
	// field is zero or negative, we read the full body.
	MaxBodySize int64 `json:"max_body_size"`
}

func (r *HTTPRequest) newRequest(ctx context.Context) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", r.URL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header = r.Headers.Clone() // copy
	httpReq.Host = r.Host
	return httpReq, nil
}

// HTTPResponse is the response corresponding to an HTTPRequest.
type HTTPResponse struct {
	// StatusCode is the response status code.
	StatusCode int `json:"status_code"`

	// Headers contains response headers.
	Headers http.Header `json:"headers"`

	// Body contains the response body.
	Body []byte `json:"body"`
}

// newHTTPClient creates a new httpClient.
func newHTTPClient(begin time.Time, txp httpTransport) *httpClient {
	return &httpClient{begin: begin, txp: txp}
}

type httpClient struct {
	begin time.Time
	txp   httpTransport
}

func (c *httpClient) Get(ctx context.Context, req *HTTPRequest) *HTTPRequestResponse {
	clnt := c.newClient(req.Cookies)
	defer clnt.CloseIdleConnections()                       // respect the protocol
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // enforce timeout
	defer cancel()
	m := &HTTPRequestResponse{
		Request: req,
		Started: time.Since(c.begin),
	}
	httpReq, err := req.newRequest(ctx)
	if err != nil {
		m.Failure = err
		return m
	}
	resp, err := clnt.Do(httpReq)
	m.Completed = time.Since(c.begin)
	if err != nil {
		m.Failure = err
		return m
	}
	defer resp.Body.Close()
	m.Response = &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}
	body, err := iox.ReadAllContext(ctx, req.bodyReadPolicy(resp.Body))
	if err != nil {
		m.Failure = err
		return m
	}
	m.Response.Body = body
	return m
}

func (req *HTTPRequest) bodyReadPolicy(r io.Reader) io.Reader {
	if req.MaxBodySize > 0 {
		return io.LimitReader(r, req.MaxBodySize)
	}
	return r
}

func (c *httpClient) newClient(jar http.CookieJar) *http.Client {
	return &http.Client{
		Transport: c.txp,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: jar,
	}
}
