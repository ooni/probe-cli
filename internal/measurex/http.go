package measurex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/publicsuffix"
)

// HTTPTransport is the HTTP transport type we use.
type HTTPTransport interface {
	netxlite.HTTPTransport

	// ConnID returns the connection ID.
	ConnID() int64
}

// WrapHTTPTransport wraps a netxlite.HTTPTransport to add measurex
// capabilities. With this constructor the conn ID is undefined.
func WrapHTTPTransport(
	origin Origin, db DB, txp netxlite.HTTPTransport) HTTPTransport {
	return WrapHTTPTransportWithConnID(origin, db, txp, 0)
}

// WrapHTTPTransportWithConnID is like WrapHTTPTransport but also
// sets the conn ID, which is otherwise undefined.
func WrapHTTPTransportWithConnID(origin Origin,
	db DB, txp netxlite.HTTPTransport, connID int64) HTTPTransport {
	return &httpTransportx{
		HTTPTransport: txp, db: db, connID: connID, origin: origin}
}

// NewHTTPTransportWithConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func NewHTTPTransportWithConn(
	origin Origin, logger Logger, db DB, conn Conn) HTTPTransport {
	return WrapHTTPTransportWithConnID(origin, db, netxlite.NewHTTPTransport(
		logger, netxlite.NewSingleUseDialer(conn),
		netxlite.NewNullTLSDialer(),
	), conn.ConnID())
}

// NewHTTPTransportWithTLSConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func NewHTTPTransportWithTLSConn(
	origin Origin, logger Logger, db DB, conn TLSConn) HTTPTransport {
	return WrapHTTPTransportWithConnID(origin, db, netxlite.NewHTTPTransport(
		logger, netxlite.NewNullDialer(),
		netxlite.NewSingleUseTLSDialer(conn),
	), conn.ConnID())
}

type httpTransportx struct {
	netxlite.HTTPTransport
	connID int64
	db     DB
	origin Origin
}

// HTTPRoundTripEvent contains information about an HTTP round trip.
//
// If ConnID is zero or negative, it means undefined. This happens
// when we create a transport without knowing the ConnID.
type HTTPRoundTripEvent struct {
	Origin               Origin
	MeasurementID        int64
	ConnID               int64
	RequestMethod        string
	RequestURL           *url.URL
	RequestHeader        http.Header
	Started              time.Time
	Finished             time.Time
	Error                error
	ResponseStatus       int
	ResponseHeader       http.Header
	ResponseBodySnapshot []byte
}

// We only read a small snapshot of the body to keep measurements
// lean, since we're mostly interested in TLS interference nowadays
// but we'll also allow for reading more bytes from the conn.
const maxBodySnapshot = 1 << 11

func (txp *httpTransportx) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Now()
	resp, err := txp.HTTPTransport.RoundTrip(req)
	rt := &HTTPRoundTripEvent{
		Origin:        txp.origin,
		MeasurementID: txp.db.MeasurementID(),
		ConnID:        txp.connID,
		RequestMethod: req.Method,
		RequestURL:    req.URL,
		RequestHeader: req.Header,
		Started:       started,
	}
	if err != nil {
		rt.Finished = time.Now()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	rt.ResponseStatus = resp.StatusCode
	rt.ResponseHeader = resp.Header
	r := io.LimitReader(resp.Body, maxBodySnapshot)
	body, err := iox.ReadAllContext(req.Context(), r)
	if errors.Is(err, io.EOF) && resp.Close {
		err = nil // we expected to see an EOF here
	}
	if err != nil {
		rt.Finished = time.Now()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	resp.Body = &httpTransportBody{ // allow for reading more if needed
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}
	rt.ResponseBodySnapshot = body
	rt.Finished = time.Now()
	txp.db.InsertIntoHTTPRoundTrip(rt)
	return resp, nil
}

type httpTransportBody struct {
	io.Reader
	io.Closer
}

func (txp *httpTransportx) ConnID() int64 {
	return txp.connID
}

// HTTPClient is the HTTP client type we use.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// NewHTTPClient creates a new HTTPClient instance that
// does not automatically perform redirects.
func NewHTTPClientWithoutRedirects(origin Origin, db DB, txp HTTPTransport) HTTPClient {
	return newHTTPClient(origin, db, txp, http.ErrUseLastResponse)
}

// NewHTTPClientWithRedirects creates a new HTTPClient
// instance that automatically perform redirects.
func NewHTTPClientWithRedirects(origin Origin, db DB, txp HTTPTransport) HTTPClient {
	return newHTTPClient(origin, db, txp, nil)
}

// HTTPRedirectEvent records an HTTP redirect.
//
// If ConnID is zero or negative, it means undefined. This happens
// when we create a transport without knowing the ConnID.
//
// The Request field contains the next request to issue. When
// redirects are disabled, this field contains the request you
// should issue to continue the redirect chain.
//
// The Via field contains the requests issued so far. The first
// request inside Via is the last one that has been issued.
//
// The Error field can have three values:
//
// - nil if the redirect occurred;
//
// - ErrHTTPTooManyRedirects when we see too many redirections;
//
// - http.ErrUseLastResponse if redirections are disabled.
type HTTPRedirectEvent struct {
	Origin        Origin
	MeasurementID int64
	ConnID        int64
	Request       *http.Request
	Via           []*http.Request
	Error         error
}

// MarshalJSON marshals an HTTPRedirectEvent to JSON.
func (ev *HTTPRedirectEvent) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"Origin":        ev.Origin,
		"MeasurementID": ev.MeasurementID,
		"ConnID":        ev.ConnID,
		"Request":       ev.simplifyRequest(ev.Request),
		"Via":           ev.simplifyRequests(ev.Via),
		"Error":         ev.Error,
	}
	return json.Marshal(m)
}

// simplifyRequest simplifies a single http.Request so
// that it could be serialized as a JSON.
func (ev *HTTPRedirectEvent) simplifyRequest(req *http.Request) (out map[string]interface{}) {
	out = map[string]interface{}{
		"URL":    req.URL,
		"Header": req.Header,
	}
	return
}

// simplifyRequests is simplifyRequest applied to a list
// of http.Request rather than just one of them.
func (ev *HTTPRedirectEvent) simplifyRequests(req []*http.Request) (out []map[string]interface{}) {
	for _, r := range req {
		out = append(out, ev.simplifyRequest(r))
	}
	return
}

// ErrHTTPTooManyRedirects is the unexported error that the standard library
// would return when hitting too many redirects.
var ErrHTTPTooManyRedirects = errors.New("stopped after 10 redirects")

func newHTTPClient(origin Origin, db DB, txp HTTPTransport, defaultErr error) HTTPClient {
	return &http.Client{
		Transport: txp,
		Jar:       NewCookieJar(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			err := defaultErr
			if len(via) >= 10 {
				err = ErrHTTPTooManyRedirects
			}
			db.InsertIntoHTTPRedirect(&HTTPRedirectEvent{
				Origin:        origin,
				MeasurementID: db.MeasurementID(),
				ConnID:        txp.ConnID(),
				Request:       req,
				Via:           via,
				Error:         err,
			})
			return err
		},
	}
}

// NewCookieJar is a convenience factory for creating an http.CookieJar
// that is aware of the effective TLS / public suffix list. This
// means that the jar won't allow a domain to set cookies for another
// unrelated domain (in the public-suffix-list sense).
func NewCookieJar() http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	// Safe to PanicOnError here: cookiejar.New _always_ returns nil.
	runtimex.PanicOnError(err, "cookiejar.New failed")
	return jar
}

// NewHTTPRequestHeaderForMeasuring returns an http.Header where
// the headers are the ones we use for measuring.
func NewHTTPRequestHeaderForMeasuring() http.Header {
	h := http.Header{}
	h.Set("Accept", httpheader.Accept())
	h.Set("Accept-Language", httpheader.AcceptLanguage())
	h.Set("User-Agent", httpheader.UserAgent())
	return h
}

// NewHTTPRequestWithContext is a convenience factory for creating
// a new HTTP request with the typical headers we use when performing
// measurements already set inside of req.Header.
func NewHTTPRequestWithContext(ctx context.Context,
	method, URL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, URL, body)
	if err != nil {
		return nil, err
	}
	req.Header = NewHTTPRequestHeaderForMeasuring()
	return req, nil
}

// NewHTTPGetRequest is a convenience factory for creating a new
// http.Request using the GET method and the given URL.
func NewHTTPGetRequest(ctx context.Context, URL string) (*http.Request, error) {
	return NewHTTPRequestWithContext(ctx, "GET", URL, nil)
}

// MustNewHTTPGetRequest is a convenience factory for creating
// a new http.Request using GET that panics on error.
func MustNewHTTPGetRequest(ctx context.Context, URL string) *http.Request {
	req, err := NewHTTPGetRequest(ctx, URL)
	runtimex.PanicOnError(err, "NewHTTPGetRequest failed")
	return req
}
