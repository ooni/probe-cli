package urlgetter

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

// TCPConnect measures a tcpconnect://<domain>:<port>/ URL.
func (rx *Runner) TCPConnect(ctx context.Context, config *Config, URL *url.URL) error {
	conn, err := rx.tcpConnect(ctx, config, URL)
	measurexlite.MaybeClose(conn)
	return err
}

func (rx *Runner) tcpConnect(ctx context.Context, config *Config, URL *url.URL) (*TCPConn, error) {
	// resolve the URL's domain using DNS
	addrs, err := rx.DNSLookupOp(ctx, config, URL)
	if err != nil {
		return nil, err
	}

	// loop until we establish a single TCP connection
	var errv []error
	for _, addr := range addrs {
		conn, err := rx.TCPConnectOp(ctx, addr)
		if err != nil {
			errv = append(errv, err)
			continue
		}
		return conn, nil
	}

	// either return a joined error or nil
	return nil, errors.Join(errv...)
}

// TCPConn is an established TCP connection.
type TCPConn struct {
	// Config is the original config.
	Config *Config

	// Conn is the conn.
	Conn net.Conn

	// Trace is the trace.
	Trace *measurexlite.Trace

	// URL is the original URL.
	URL *url.URL
}

var _ io.Closer = &TCPConn{}

// AsHTTPConn converts a [*TCPConn] to an [*HTTPConn].
func (cx *TCPConn) AsHTTPConn(logger model.Logger) *HTTPConn {
	return &HTTPConn{
		Config:                cx.Config,
		Conn:                  cx.Conn,
		Network:               "tcp",
		RemoteAddress:         measurexlite.SafeRemoteAddrString(cx.Conn),
		TLSNegotiatedProtocol: "",
		Trace:                 cx.Trace,
		Transport: netxlite.NewHTTPTransportWithOptions(
			logger,
			netxlite.NewSingleUseDialer(cx.Conn),
			netxlite.NewNullTLSDialer(),
		),
		URL: cx.URL,
	}
}

// Close implements io.Closer.
func (tx *TCPConn) Close() error {
	return tx.Conn.Close()
}

// TCPConnectOp establishes a TCP connection.
func (rx *Runner) TCPConnectOp(ctx context.Context, input *DNSLookupResult) (*TCPConn, error) {
	// enforce timeout
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain the next trace index
	index := rx.IndexGen.Next()

	// create trace using the given underlying network
	trace := measurexlite.NewTrace(index, rx.Begin)
	trace.Netx = &netxlite.Netx{Underlying: rx.UNet}

	// obtain logger
	logger := rx.Session.Logger()

	// create dialer
	dialer := trace.NewDialerWithoutResolver(logger)

	// the endpoint to use depends on the DNS lookup results
	endpoint, err := input.endpoint()
	if err != nil {
		return nil, err
	}

	// start operation logger
	ol := logx.NewOperationLogger(logger, "[#%d] TCP connect %s", trace.Index(), endpoint)

	// establish the TCP connection
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)

	// stop the operation logger
	ol.Stop(err)

	// append the TCP connect results
	rx.TestKeys.AppendTCPConnect(trace.TCPConnects()...)

	// append the network events caused by TCP connect
	rx.TestKeys.AppendNetworkEvents(trace.NetworkEvents()...)

	// in case of failure, set failed operation and failure
	if err != nil {
		rx.TestKeys.MaybeSetFailedOperation(netxlite.ConnectOperation)
		rx.TestKeys.MaybeSetFailure(err.Error())
		return nil, err
	}

	// start measuring throttling using a sampler
	sampler := throttling.NewSampler(trace)

	// return the result
	result := &TCPConn{
		Config: input.Config,
		Conn: &tcpConnWrapper{
			Conn:     conn,
			Once:     &sync.Once{},
			Sampler:  sampler,
			TestKeys: rx.TestKeys,
			Trace:    trace,
		},
		Trace: trace,
		URL:   input.URL,
	}
	return result, nil
}

// tcpConnWrapper wraps a connection and saves network events.
type tcpConnWrapper struct {
	net.Conn
	Once     *sync.Once
	Sampler  *throttling.Sampler
	TestKeys RunnerTestKeys
	Trace    *measurexlite.Trace
}

// Close implements [io.Closer].
func (c *tcpConnWrapper) Close() (err error) {
	c.Once.Do(func() {
		err = c.Conn.Close()
		c.TestKeys.AppendNetworkEvents(c.Trace.NetworkEvents()...)
		c.TestKeys.AppendNetworkEvents(c.Sampler.ExtractSamples()...)
		_ = c.Sampler.Close()
	})
	return
}
