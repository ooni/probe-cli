package tracex

//
// HTTP
//

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// MaybeWrapHTTPTransport wraps the HTTPTransport to save events if this Saver
// is not nil and otherwise just returns the given HTTPTransport. The snapshotSize
// argument is the maximum response body snapshot size to save per response.
func (s *Saver) MaybeWrapHTTPTransport(txp model.HTTPTransport, snapshotSize int64) model.HTTPTransport {
	if s != nil {
		txp = &HTTPTransportSaver{
			HTTPTransport: txp,
			Saver:         s,
			SnapshotSize:  snapshotSize,
		}
	}
	return txp
}

// httpCloneRequestHeaders returns a clone of the headers where we have
// also set the host header, which normally is not set by
// golang until it serializes the request itself.
func httpCloneRequestHeaders(req *http.Request) http.Header {
	header := req.Header.Clone()
	if req.Host != "" {
		header.Set("Host", req.Host)
	} else {
		header.Set("Host", req.URL.Host)
	}
	return header
}

// HTTPTransportSaver is a RoundTripper that saves
// events related to the HTTP transaction
type HTTPTransportSaver struct {
	// HTTPTransport is the MANDATORY underlying HTTP transport.
	HTTPTransport model.HTTPTransport

	// Saver is the MANDATORY saver to use.
	Saver *Saver

	// SnapshotSize is the OPTIONAL maximum body snapshot size (if not set, we'll
	// use 1<<17, which we've been using since the ooni/netx days)
	SnapshotSize int64
}

// HTTPRoundTrip performs the round trip with the given transport and
// the given arguments and saves the results into the saver.
//
// The maxBodySnapshotSize argument controls the maximum size of the
// body snapshot that we collect along with the HTTP round trip.
func (txp *HTTPTransportSaver) RoundTrip(req *http.Request) (*http.Response, error) {

	started := time.Now()
	txp.Saver.Write(&EventHTTPTransactionStart{&EventValue{
		HTTPRequestHeaders: httpCloneRequestHeaders(req),
		HTTPMethod:         req.Method,
		HTTPURL:            req.URL.String(),
		Transport:          txp.HTTPTransport.Network(),
		Time:               started,
	}})
	ev := &EventValue{
		HTTPRequestHeaders: httpCloneRequestHeaders(req),
		HTTPMethod:         req.Method,
		HTTPURL:            req.URL.String(),
		Transport:          txp.HTTPTransport.Network(),
		Time:               time.Time{}, // see below
	}
	defer func() {
		ev.Time = time.Now() // https://github.com/ooni/probe/issues/2137
		txp.Saver.Write(&EventHTTPTransactionDone{ev})
	}()

	resp, err := txp.HTTPTransport.RoundTrip(req)

	if err != nil {
		ev.Duration = time.Since(started)
		ev.Err = NewFailureStr(err)
		return nil, err
	}

	ev.HTTPStatusCode = resp.StatusCode
	ev.HTTPResponseHeaders = resp.Header.Clone()

	maxBodySnapshotSize := txp.snapshotSize()
	r := io.LimitReader(resp.Body, maxBodySnapshotSize)
	body, err := netxlite.ReadAllContext(req.Context(), r)

	if err != nil {
		ev.Duration = time.Since(started)
		ev.Err = NewFailureStr(err)
		return nil, err
	}

	resp.Body = &httpReadableAgainBody{ // allow for reading again the whole body
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}

	ev.Duration = time.Since(started)
	ev.HTTPResponseBody = body
	ev.HTTPResponseBodyIsTruncated = int64(len(body)) >= maxBodySnapshotSize

	return resp, nil
}

func (txp *HTTPTransportSaver) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

func (txp *HTTPTransportSaver) Network() string {
	return txp.HTTPTransport.Network()
}

func (txp *HTTPTransportSaver) snapshotSize() int64 {
	if txp.SnapshotSize > 0 {
		return txp.SnapshotSize
	}
	return 1 << 17
}

type httpReadableAgainBody struct {
	io.Reader
	io.Closer
}

var _ model.HTTPTransport = &HTTPTransportSaver{}
