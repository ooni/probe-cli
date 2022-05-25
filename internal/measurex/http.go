package measurex

//
// HTTP
//
// This file contains basic networking code. We provide:
//
// - a wrapper for netxlite.HTTPTransport that stores
// round trip events into an EventDB
//
// - an interface that is http.Client like and one internal
// implementation of such an interface that helps us to
// store HTTP redirections info into an EventDB
//

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
	"unicode/utf8"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/publicsuffix"
)

// WrapHTTPTransport creates a new transport that saves
// HTTP events into the WritableDB.
func (mx *Measurer) WrapHTTPTransport(
	db WritableDB, txp model.HTTPTransport) *HTTPTransportDB {
	return WrapHTTPTransport(mx.Begin, db, txp, mx.httpMaxBodySnapshotSize())
}

// DefaultHTTPMaxBodySnapshotSize is the default size used when
// saving HTTP body snapshots. We only save a small snapshot of the
// body to keep measurements lean, since we're mostly interested
// in TLS interference nowadays and much less in full bodies.
const DefaultHTTPMaxBodySnapshotSize = 1 << 11

// httpMaxBodySnapshotSize selects the maximum body snapshot size.
func (mx *Measurer) httpMaxBodySnapshotSize() int64 {
	if mx.HTTPMaxBodySnapshotSize > 0 {
		return mx.HTTPMaxBodySnapshotSize
	}
	return DefaultHTTPMaxBodySnapshotSize
}

// WrapHTTPTransport creates a new model.HTTPTransport instance
// using the following configuration:
//
// - begin is the conventional "zero time" indicating the
// moment when the measurement begun;
//
// - db is the writable DB into which to write the measurement;
//
// - txp is the underlying transport to use;
//
// - maxBodySnapshotSize is the max size of the response body snapshot
// to save: we'll truncate bodies larger than that.
func WrapHTTPTransport(
	begin time.Time, db WritableDB, txp model.HTTPTransport,
	maxBodySnapshotSize int64) *HTTPTransportDB {
	return &HTTPTransportDB{
		HTTPTransport:       txp,
		Begin:               begin,
		DB:                  db,
		MaxBodySnapshotSize: maxBodySnapshotSize,
	}
}

// NewHTTPTransportWithConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func (mx *Measurer) NewHTTPTransportWithConn(
	logger model.Logger, db WritableDB, conn Conn) *HTTPTransportDB {
	return mx.WrapHTTPTransport(db, netxlite.NewHTTPTransport(
		logger, netxlite.NewSingleUseDialer(conn), netxlite.NewNullTLSDialer()))
}

// NewHTTPTransportWithTLSConn creates and wraps an HTTPTransport that
// does not dial and only uses the given conn.
func (mx *Measurer) NewHTTPTransportWithTLSConn(
	logger model.Logger, db WritableDB, conn netxlite.TLSConn) *HTTPTransportDB {
	return mx.WrapHTTPTransport(db, netxlite.NewHTTPTransport(
		logger, netxlite.NewNullDialer(), netxlite.NewSingleUseTLSDialer(conn)))
}

// NewHTTPTransportWithQUICConn creates and wraps an HTTPTransport that
// does not dial and only uses the given QUIC connection.
func (mx *Measurer) NewHTTPTransportWithQUICConn(
	logger model.Logger, db WritableDB, qconn quic.EarlyConnection) *HTTPTransportDB {
	return mx.WrapHTTPTransport(db, netxlite.NewHTTP3Transport(
		logger, netxlite.NewSingleUseQUICDialer(qconn), &tls.Config{}))
}

// HTTPTransportDB is an implementation of HTTPTransport that
// writes measurement events into a WritableDB.
//
// There are many factories to construct this data type. Otherwise,
// you can construct it manually. In which case, do not modify
// public fields during usage, since this may cause a data race.
type HTTPTransportDB struct {
	model.HTTPTransport

	// Begin is when we started measuring.
	Begin time.Time

	// DB is where to write events.
	DB WritableDB

	// MaxBodySnapshotSize is the maximum size of the body
	// snapshot that we take during a round trip.
	MaxBodySnapshotSize int64
}

// HTTPRequest is the HTTP request.
type HTTPRequest struct {
	// Names consistent with df-001-http.md
	Method  string          `json:"method"`
	URL     string          `json:"url"`
	Headers ArchivalHeaders `json:"headers"`
}

// HTTPResponse is the HTTP response.
type HTTPResponse struct {
	// Names consistent with df-001-http.md
	Code            int64               `json:"code"`
	Headers         ArchivalHeaders     `json:"headers"`
	Body            *ArchivalBinaryData `json:"body"`
	BodyIsTruncated bool                `json:"body_is_truncated"`

	// Fields not part of the spec
	BodyLength int64 `json:"x_body_length"`
	BodyIsUTF8 bool  `json:"x_body_is_utf8"`
}

// HTTPRoundTripEvent contains information about an HTTP round trip.
type HTTPRoundTripEvent struct {
	Failure                 *string
	Method                  string
	URL                     string
	RequestHeaders          http.Header
	StatusCode              int64
	ResponseHeaders         http.Header
	ResponseBody            []byte
	ResponseBodyLength      int64
	ResponseBodyIsTruncated bool
	ResponseBodyIsUTF8      bool
	Finished                float64
	Started                 float64
	Oddity                  Oddity
}

func (txp *HTTPTransportDB) RoundTrip(req *http.Request) (*http.Response, error) {
	started := time.Since(txp.Begin).Seconds()
	resp, err := txp.HTTPTransport.RoundTrip(req)
	rt := &HTTPRoundTripEvent{
		Method:         req.Method,
		URL:            req.URL.String(),
		RequestHeaders: req.Header,
		Started:        started,
	}
	if err != nil {
		rt.Finished = time.Since(txp.Begin).Seconds()
		rt.Failure = NewFailure(err)
		txp.DB.InsertIntoHTTPRoundTrip(rt)
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
	rt.StatusCode = int64(resp.StatusCode)
	rt.ResponseHeaders = resp.Header
	r := io.LimitReader(resp.Body, txp.MaxBodySnapshotSize)
	body, err := netxlite.ReadAllContext(req.Context(), r)
	if err != nil {
		rt.Finished = time.Since(txp.Begin).Seconds()
		rt.Failure = NewFailure(err)
		txp.DB.InsertIntoHTTPRoundTrip(rt)
		return nil, err
	}
	resp.Body = &httpTransportBody{ // allow for reading more if needed
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}
	rt.ResponseBody = body
	rt.ResponseBodyLength = int64(len(body))
	rt.ResponseBodyIsTruncated = int64(len(body)) >= txp.MaxBodySnapshotSize
	rt.ResponseBodyIsUTF8 = utf8.Valid(body)
	rt.Finished = time.Since(txp.Begin).Seconds()
	txp.DB.InsertIntoHTTPRoundTrip(rt)
	return resp, nil
}

type httpTransportBody struct {
	io.Reader
	io.Closer
}

// NewHTTPClient creates a new HTTPClient instance that
// does not automatically perform redirects.
func NewHTTPClientWithoutRedirects(
	db WritableDB, jar http.CookieJar, txp model.HTTPTransport) model.HTTPClient {
	return newHTTPClient(db, jar, txp, http.ErrUseLastResponse)
}

// NewHTTPClientWithRedirects creates a new HTTPClient
// instance that automatically perform redirects.
func NewHTTPClientWithRedirects(
	db WritableDB, jar http.CookieJar, txp model.HTTPTransport) model.HTTPClient {
	return newHTTPClient(db, jar, txp, nil)
}

// HTTPRedirectEvent records an HTTP redirect.
type HTTPRedirectEvent struct {
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

func newHTTPClient(db WritableDB, cookiejar http.CookieJar,
	txp model.HTTPTransport, defaultErr error) model.HTTPClient {
	return netxlite.WrapHTTPClient(&http.Client{
		Transport: txp,
		Jar:       cookiejar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			err := defaultErr
			if len(via) >= 10 {
				err = ErrHTTPTooManyRedirects
			}
			db.InsertIntoHTTPRedirect(&HTTPRedirectEvent{
				URL:      via[0].URL, // bug in Go stdlib if we crash here
				Location: req.URL,
				Cookies:  cookiejar.Cookies(req.URL),
				Error:    err,
			})
			return err
		},
	})
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
	h.Set("Accept", model.HTTPHeaderAccept)
	h.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
	h.Set("User-Agent", model.HTTPHeaderUserAgent)
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
