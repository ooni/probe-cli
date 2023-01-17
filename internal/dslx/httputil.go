package dslx

import (
	"context"
	"errors"
	"sync/atomic"
)

// HTTPJustUseOneConn is a filter that allows the first connection that
// reaches this stage to make progress and stops subsequent ones.
func HTTPJustUseOneConn() Func[*HTTPTransport, *Maybe[*HTTPTransport]] {
	return &httpJustUseOneConnFunc{
		counter: &atomic.Int64{},
	}
}

// httpJustUseOneConnFunc is the function returned by HTTPJustUseOneConn
type httpJustUseOneConnFunc struct {
	counter *atomic.Int64
}

// ErrHTTPSubsequentConn indicates that this connection was prevented from
// measuring because a previous connection already completed.
var ErrHTTPSubsequentConn = errors.New("dslx: subsequent HTTP conn")

// Apply implements Func
func (f *httpJustUseOneConnFunc) Apply(
	ctx context.Context, state *HTTPTransport) *Maybe[*HTTPTransport] {
	return &Maybe[*HTTPTransport]{
		Error:        nil,
		Observations: nil,
		Skipped:      f.counter.Add(1) > 1,
		State:        state,
	}
}
