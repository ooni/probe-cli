package archival

//
// Saves HTTP events
//

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPRoundTripEvent contains an HTTP round trip.
type HTTPRoundTripEvent struct {
	Failure                 error
	Finished                time.Time
	Method                  string
	RequestHeaders          http.Header
	ResponseBody            []byte
	ResponseBodyIsTruncated bool
	ResponseBodyLength      int64
	ResponseHeaders         http.Header
	Started                 time.Time
	StatusCode              int64
	Transport               string
	URL                     string
}

// HTTPRoundTrip performs the round trip with the given transport and
// the given arguments and saves the results into the saver.
//
// The maxBodySnapshotSize argument controls the maximum size of the
// body snapshot that we collect along with the HTTP round trip.
func (s *Saver) HTTPRoundTrip(
	txp model.HTTPTransport, maxBodySnapshotSize int64,
	req *http.Request) (*http.Response, error) {
	started := time.Now()
	resp, err := txp.RoundTrip(req)
	rt := &HTTPRoundTripEvent{
		Failure:                 nil,         // set later
		Finished:                time.Time{}, // set later
		Method:                  req.Method,
		RequestHeaders:          req.Header.Clone(),
		ResponseBody:            nil, // set later
		ResponseBodyIsTruncated: false,
		ResponseBodyLength:      0,
		ResponseHeaders:         nil, // set later
		Started:                 started,
		StatusCode:              0, // set later
		Transport:               txp.Network(),
		URL:                     req.URL.String(),
	}
	if err != nil {
		rt.Finished = time.Now()
		rt.Failure = err
		s.appendHTTPRoundTripEvent(rt)
		return nil, err
	}
	rt.StatusCode = int64(resp.StatusCode)
	rt.ResponseHeaders = resp.Header.Clone()
	r := io.LimitReader(resp.Body, maxBodySnapshotSize)
	body, err := netxlite.ReadAllContext(req.Context(), r)
	if err != nil {
		rt.Finished = time.Now()
		rt.Failure = err
		s.appendHTTPRoundTripEvent(rt)
		return nil, err
	}
	resp.Body = &archivalHTTPTransportBody{ // allow for reading more if needed
		Reader: io.MultiReader(bytes.NewReader(body), resp.Body),
		Closer: resp.Body,
	}
	rt.ResponseBody = body
	rt.ResponseBodyLength = int64(len(body))
	rt.ResponseBodyIsTruncated = int64(len(body)) >= maxBodySnapshotSize
	rt.Finished = time.Now()
	s.appendHTTPRoundTripEvent(rt)
	return resp, nil
}

type archivalHTTPTransportBody struct {
	io.Reader
	io.Closer
}

func (s *Saver) appendHTTPRoundTripEvent(ev *HTTPRoundTripEvent) {
	s.mu.Lock()
	s.trace.HTTPRoundTrip = append(s.trace.HTTPRoundTrip, ev)
	s.mu.Unlock()
}
