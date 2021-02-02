package oldhttptransport

import (
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/transactionid"
)

// BodyTracer performs single HTTP transactions and emits
// measurement events as they happen.
type BodyTracer struct {
	Transport http.RoundTripper
}

// NewBodyTracer creates a new Transport.
func NewBodyTracer(roundTripper http.RoundTripper) *BodyTracer {
	return &BodyTracer{Transport: roundTripper}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *BodyTracer) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.Transport.RoundTrip(req)
	if err != nil {
		return
	}
	// "The http Client and Transport guarantee that Body is always
	//  non-nil, even on responses without a body or responses with
	//  a zero-length body." (from the docs)
	resp.Body = &bodyWrapper{
		ReadCloser: resp.Body,
		root:       modelx.ContextMeasurementRootOrDefault(req.Context()),
		tid:        transactionid.ContextTransactionID(req.Context()),
	}
	return
}

// CloseIdleConnections closes the idle connections.
func (t *BodyTracer) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.Transport.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}

type bodyWrapper struct {
	io.ReadCloser
	root *modelx.MeasurementRoot
	tid  int64
}

func (bw *bodyWrapper) Read(b []byte) (n int, err error) {
	n, err = bw.ReadCloser.Read(b)
	bw.root.Handler.OnMeasurement(modelx.Measurement{
		HTTPResponseBodyPart: &modelx.HTTPResponseBodyPartEvent{
			// "Read reads up to len(p) bytes into p. It returns the number of
			// bytes read (0 <= n <= len(p)) and any error encountered."
			Data:                   b[:n],
			Error:                  err,
			DurationSinceBeginning: time.Now().Sub(bw.root.Beginning),
			TransactionID:          bw.tid,
		},
	})
	return
}

func (bw *bodyWrapper) Close() (err error) {
	err = bw.ReadCloser.Close()
	bw.root.Handler.OnMeasurement(modelx.Measurement{
		HTTPResponseDone: &modelx.HTTPResponseDoneEvent{
			DurationSinceBeginning: time.Now().Sub(bw.root.Beginning),
			TransactionID:          bw.tid,
		},
	})
	return
}
