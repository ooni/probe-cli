package measurex

import (
	"bytes"
	"context"
	"crypto/tls"
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

// HTTPTransport is the HTTP transport type we use. This transport
// is a normal netxlite.HTTPTransport but also knows about the ConnID.
//
// The RoundTrip method of this transport MAY read a small snapshot
// of the response body to include it into the measurement. When this
// happens, the transport will nonetheless return a response body
// that is suitable for reading the whole body again. The only difference
// with reading the body normally is timing. The snapshot will be read
// immediately because it's already cached in RAM. The rest of the
// body instead will be read normally, using the network.
type HTTPTransport interface {
	netxlite.HTTPTransport

	// ConnID returns the connection ID. When this value is zero
	// or negative it means it has not been set.
	ConnID() int64
}

// WrapHTTPTransport takes in input a netxlite.HTTPTransport and
// returns an HTTPTransport that uses the DB to save events occurring
// during HTTP round trips. With this constructor the ConnID is
// not set, hence ConnID will always return zero.
func WrapHTTPTransport(measurementID int64,
	origin Origin, db EventDB, txp netxlite.HTTPTransport) HTTPTransport {
	return WrapHTTPTransportWithConnID(measurementID, origin, db, txp, 0)
}

// WrapHTTPTransportWithConnID is like WrapHTTPTransport but also
// sets the conn ID, which is otherwise set to zero.
func WrapHTTPTransportWithConnID(measurementID int64, origin Origin,
	db EventDB, txp netxlite.HTTPTransport, connID int64) HTTPTransport {
	return &httpTransportx{
		HTTPTransport: txp,
		db:            db,
		connID:        connID,
		mid:           measurementID,
		origin:        origin,
	}
}

// NewHTTPTransportWithConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func NewHTTPTransportWithConn(measurementID int64,
	origin Origin, logger Logger, db EventDB, conn Conn) HTTPTransport {
	txp := netxlite.NewHTTPTransport(logger, netxlite.NewSingleUseDialer(conn),
		netxlite.NewNullTLSDialer())
	return WrapHTTPTransportWithConnID(
		measurementID, origin, db, txp, conn.ConnID())
}

// NewHTTPTransportWithTLSConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func NewHTTPTransportWithTLSConn(measurementID int64,
	origin Origin, logger Logger, db EventDB, conn TLSConn) HTTPTransport {
	txp := netxlite.NewHTTPTransport(logger, netxlite.NewNullDialer(),
		netxlite.NewSingleUseTLSDialer(conn))
	return WrapHTTPTransportWithConnID(
		measurementID, origin, db, txp, conn.ConnID())
}

// NewHTTPTransportWithQUICSess creates and wraps an HTTPTransport that
// does not dial and only uses the given QUIC session.
func NewHTTPTransportWithQUICSess(measurementID int64,
	origin Origin, logger Logger, db EventDB, sess QUICEarlySession) HTTPTransport {
	txp := netxlite.NewHTTP3Transport(
		logger, netxlite.NewSingleUseQUICDialer(sess), &tls.Config{})
	return WrapHTTPTransportWithConnID(
		measurementID, origin, db, txp, sess.ConnID())
}

type httpTransportx struct {
	netxlite.HTTPTransport
	connID int64
	db     EventDB
	mid    int64
	origin Origin
}

// HTTPRoundTripEvent contains information about an HTTP round trip.
//
// If ConnID is zero or negative, it means undefined. This happens
// when we create a transport without knowing the ConnID.
type HTTPRoundTripEvent struct {
	Origin               Origin        // OriginProbe or OriginTH
	MeasurementID        int64         // ID of the measurement
	ConnID               int64         // ID of the conn (<= zero means undefined)
	RequestMethod        string        // Request method
	RequestURL           *url.URL      // Request URL
	RequestHeader        http.Header   // Request headers
	Started              time.Duration // Beginning of round trip
	Finished             time.Duration // End of round trip
	Error                error         // Error or nil
	Oddity               Oddity        // Oddity classification
	ResponseStatus       int           // Status code
	ResponseHeader       http.Header   // Response headers
	ResponseBodySnapshot []byte        // Body snapshot
	MaxBodySnapshotSize  int64         // Max size for snapshot
}

// We only read a small snapshot of the body to keep measurements
// lean, since we're mostly interested in TLS interference nowadays
// but we'll also allow for reading more bytes from the conn.
const maxBodySnapshot = 1 << 11

func (txp *httpTransportx) RoundTrip(req *http.Request) (*http.Response, error) {
	started := txp.db.ElapsedTime()
	resp, err := txp.HTTPTransport.RoundTrip(req)
	rt := &HTTPRoundTripEvent{
		Origin:              txp.origin,
		MeasurementID:       txp.mid,
		ConnID:              txp.connID,
		RequestMethod:       req.Method,
		RequestURL:          req.URL,
		RequestHeader:       req.Header,
		Started:             started,
		MaxBodySnapshotSize: maxBodySnapshot,
	}
	if err != nil {
		rt.Finished = txp.db.ElapsedTime()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	switch {
	case resp.StatusCode == 403:
		rt.Oddity = OddityStatus403
	case resp.StatusCode == 404:
		rt.Oddity = OddityStatus404
	case resp.StatusCode == 503:
		rt.Oddity = OddityStatus503
	case resp.StatusCode >= 400:
		rt.Oddity = OddityStatusOther
	}
	rt.ResponseStatus = resp.StatusCode
	rt.ResponseHeader = resp.Header
	r := io.LimitReader(resp.Body, maxBodySnapshot)
	body, err := iox.ReadAllContext(req.Context(), r)
	if errors.Is(err, io.EOF) && resp.Close {
		err = nil // we expected to see an EOF here, so no real error
	}
	if err != nil {
		rt.Finished = txp.db.ElapsedTime()
		rt.Error = err
		txp.db.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	resp.Body = &httpTransportBody{ // allow for reading more if needed
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}
	rt.ResponseBodySnapshot = body
	rt.Finished = txp.db.ElapsedTime()
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

// HTTPClient is the HTTP client type we use. This interface is
// compatible with http.Client. What changes in this kind of clients
// is that we'll insert redirection events into the DB.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// NewHTTPClient creates a new HTTPClient instance that
// does not automatically perform redirects.
func NewHTTPClientWithoutRedirects(measurementID int64,
	origin Origin, db EventDB, jar http.CookieJar, txp HTTPTransport) HTTPClient {
	return newHTTPClient(
		measurementID, origin, db, jar, txp, http.ErrUseLastResponse)
}

// NewHTTPClientWithRedirects creates a new HTTPClient
// instance that automatically perform redirects.
func NewHTTPClientWithRedirects(measurementID int64,
	origin Origin, db EventDB, jar http.CookieJar, txp HTTPTransport) HTTPClient {
	return newHTTPClient(
		measurementID, origin, db, jar, txp, nil)
}

// HTTPRedirectEvent records an HTTP redirect.
type HTTPRedirectEvent struct {
	// Origin is the event origin ("probe" or "th")
	Origin Origin

	// MeasurementID is the measurement inside which
	// this event occurred.
	MeasurementID int64

	// ConnID is the ID of the connection we are using,
	// which may be zero if undefined.
	ConnID int64

	// URL is the URL triggering the redirect.
	URL *url.URL

	// Location is the URL to which we're redirected.
	Location *url.URL

	// Cookies contains the cookies for Location.
	Cookies []*http.Cookie

	// The Error field can have three values:
	//
	// - nil if the redirect occurred;
	//
	// - ErrHTTPTooManyRedirects when we see too many redirections;
	//
	// - http.ErrUseLastResponse if redirections are disabled.
	Error error
}

// ErrHTTPTooManyRedirects is the unexported error that the standard library
// would return when hitting too many redirects.
var ErrHTTPTooManyRedirects = errors.New("stopped after 10 redirects")

func newHTTPClient(measurementID int64, origin Origin, db EventDB,
	cookiejar http.CookieJar, txp HTTPTransport, defaultErr error) HTTPClient {
	return &http.Client{
		Transport: txp,
		Jar:       cookiejar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			err := defaultErr
			if len(via) >= 10 {
				err = ErrHTTPTooManyRedirects
			}
			db.InsertIntoHTTPRedirect(&HTTPRedirectEvent{
				Origin:        origin,
				MeasurementID: measurementID,
				ConnID:        txp.ConnID(),
				URL:           via[0].URL, // bug in Go stdlib if we crash here
				Location:      req.URL,
				Cookies:       cookiejar.Cookies(req.URL),
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
