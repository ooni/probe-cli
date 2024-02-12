package dslvm

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

// HTTPConnection is the connection type expected by [*HTTPRoundTripStage].
type HTTPConnection interface {
	// AsSingleUseTransport converts the connection to a single-use HTTP transport
	AsSingleUseTransport(logger model.Logger) model.HTTPTransport

	// Closer embeds the Closer interface
	Closer

	// Network returns the network
	Network() string

	// RemoteAddress returns the remote address
	RemoteAddress() string

	// Scheme returns the HTTP scheme for this connection
	Scheme() string

	// TLSNegotiatedProtocol is the protocol negotiated by TLS
	TLSNegotiatedProtocol() string

	// Trace returns the Trace to use
	Trace() Trace
}

// HTTPRoundTripStage performs HTTP round trips with connections of type T.
type HTTPRoundTripStage[T HTTPConnection] struct {
	// Accept contains the OPTIONAL accept header.
	Accept string

	// AcceptLanguage contains the OPTIONAL accept-language header.
	AcceptLanguage string

	// Host contains the MANDATORY host header.
	Host string

	// Input contains the MANDATORY channel from which to connections. We
	// assume that this channel will be closed when done.
	Input <-chan T

	// MaxBodySnapshotSize is the OPTIONAL maximum body snapshot size.
	MaxBodySnapshotSize int64

	// Method contains the MANDATORY method.
	Method string

	// Output is the MANDATORY channel emitting [Void]. We will close this
	// channel when the Input channel has been closed.
	Output chan<- Done

	// Referer contains the OPTIONAL referer header.
	Referer string

	// URLPath contains the MANDATORY URL path.
	URLPath string

	// UserAgent contains the OPTIONAL user-agent header.
	UserAgent string
}

// Run is like [*TCPConnect.Run] except that it reads connections in Input and
// emits [Void] in Output. Each HTTP round trip runs in its own background
// goroutine. The parallelism is controlled by the [Runtime] ActiveConnections
// [Semaphore]. Note that this code TAKES OWNERSHIP of the connection it
// reads and closes it at the end of the round trip. While closing the conn,
// we signal [Runtime] ActiveConnections to unblock another measurement.
func (sx *HTTPRoundTripStage[T]) Run(ctx context.Context, rtx Runtime) {
	// make sure we close the output channel
	defer close(sx.Output)

	// track the number of running goroutines
	waitGroup := &sync.WaitGroup{}

	for conn := range sx.Input {
		// process connection in a background goroutine, which is fine
		// because the previous step has acquired the semaphore.
		waitGroup.Add(1)
		go func(conn HTTPConnection) {
			defer waitGroup.Done()
			defer conn.Close(rtx.Logger())         // as documented, close when done
			defer rtx.ActiveConnections().Signal() // unblock the next goroutine
			sx.roundTrip(ctx, rtx, conn)
		}(conn)
	}

	// wait for pending work to finish
	waitGroup.Wait()
}

func (sx *HTTPRoundTripStage[T]) roundTrip(ctx context.Context, rtx Runtime, conn HTTPConnection) {
	// setup
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// create HTTP request
	req, err := sx.newHTTPRequest(ctx, conn, rtx.Logger())
	if err != nil {
		return
	}

	// start the operation logger
	ol := logx.NewOperationLogger(
		rtx.Logger(),
		"[#%d] HTTPRequest %s with %s/%s host=%s",
		conn.Trace().Index(),
		req.URL.String(),
		conn.RemoteAddress(),
		conn.Network(),
		req.Host,
	)

	// perform HTTP round trip and collect observations
	observations, err := sx.doRoundTrip(ctx, conn, rtx.Logger(), req)

	// stop the operation logger
	ol.Stop(err)

	// merge and save observations
	observations = append(observations, maybeTraceToObservations(conn.Trace())...)
	rtx.SaveObservations(observations...)
}

func (sx *HTTPRoundTripStage[T]) newHTTPRequest(
	ctx context.Context, conn HTTPConnection, logger model.Logger) (*http.Request, error) {
	// create the default HTTP request
	URL := &url.URL{
		Scheme:      conn.Scheme(),
		Opaque:      "",
		User:        nil,
		Host:        sx.Host,
		Path:        sx.URLPath,
		RawPath:     "",
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}
	req, err := http.NewRequestWithContext(ctx, sx.Method, URL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Go would use URL.Host as "Host" header anyways in case we leave req.Host empty.
	// We already set it here so that we can use req.Host for logging.
	req.Host = URL.Host

	// conditionally apply headers
	if sx.Accept != "" {
		req.Header.Set("Accept", sx.Accept)
	}
	if sx.AcceptLanguage != "" {
		req.Header.Set("Accept-Language", sx.AcceptLanguage)
	}
	if sx.Referer != "" {
		req.Header.Set("Referer", sx.Referer)
	}
	if sx.UserAgent != "" {
		req.Header.Set("User-Agent", sx.UserAgent)
	}

	// req.Header["Host"] is ignored by Go but we want to have it in the measurement
	// to reflect what we think has been sent as HTTP headers.
	req.Header.Set("Host", req.Host)
	return req, nil
}

func (sx *HTTPRoundTripStage[T]) doRoundTrip(ctx context.Context,
	conn HTTPConnection, logger model.Logger, req *http.Request) ([]*Observations, error) {
	maxbody := sx.MaxBodySnapshotSize
	if maxbody < 0 {
		maxbody = 0
	}

	started := conn.Trace().TimeSince(conn.Trace().ZeroTime())

	// manually create a single 1-length observations structure because
	// the trace cannot automatically capture HTTP events
	observations := []*Observations{
		NewObservations(),
	}

	// TODO(bassosimone): https://github.com/ooni/probe-cli/pull/1505/files#r1486204572
	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			conn.Trace().Index(),
			started,
			"http_transaction_start",
			conn.Trace().Tags()...,
		))

	txp := conn.AsSingleUseTransport(logger)

	resp, err := txp.RoundTrip(req)
	var body []byte
	if err == nil {
		defer resp.Body.Close()

		// TODO(bassosimone): we should probably start sampling when
		// we create the connection rather than here

		// create sampler for measuring throttling
		sampler := throttling.NewSampler(conn.Trace())
		defer sampler.Close()

		// read a snapshot of the response body
		reader := io.LimitReader(resp.Body, maxbody)
		body, err = netxlite.ReadAllContext(ctx, reader) // TODO(https://github.com/ooni/probe/issues/2622)

		// collect and save download speed samples
		samples := sampler.ExtractSamples()
		observations[0].NetworkEvents = append(observations[0].NetworkEvents, samples...)
	}
	finished := conn.Trace().TimeSince(conn.Trace().ZeroTime())

	// TODO(bassosimone): https://github.com/ooni/probe-cli/pull/1505/files#r1486204572
	observations[0].NetworkEvents = append(observations[0].NetworkEvents,
		measurexlite.NewAnnotationArchivalNetworkEvent(
			conn.Trace().Index(),
			finished,
			"http_transaction_done",
			conn.Trace().Tags()...,
		))

	observations[0].Requests = append(observations[0].Requests,
		measurexlite.NewArchivalHTTPRequestResult(
			conn.Trace().Index(),
			started,
			conn.Network(),
			conn.RemoteAddress(),
			conn.TLSNegotiatedProtocol(),
			txp.Network(),
			req,
			resp,
			maxbody,
			body,
			err,
			finished,
			conn.Trace().Tags()...,
		))

	return observations, err
}
