package pdsl

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPResponse is the response produced by the [HTTPRoundTripTCP],
// the [HTTPRoundTripTLS], or the [HTTPRoundTripQUIC] [Filter].
type HTTPResponse struct {
	BodySnap []byte
	Resp     *http.Response
	Trace    Trace
}

// HTTPRoundTripTCP returns a [Filter] that attempts an HTTP round trip using the given [TCPConn].
func HTTPRoundTripTCP(ctx context.Context, rt Runtime, req *http.Request) Filter[TCPConn, HTTPResponse] {
	return startFilterService(func(conn TCPConn) (HTTPResponse, error) {
		dialer := netxlite.NewSingleUseDialer(conn)
		tlsDialer := netxlite.NewNullTLSDialer()
		options := []netxlite.HTTPTransportOption{ /* empty */ }
		txp := netxlite.NewHTTPTransportWithOptions(rt.Logger(), dialer, tlsDialer, options...)
		return httpRoundTrip(ctx, rt, txp, req, conn.Trace, "tcp", conn.RemoteAddr().String())
	})
}

// HTTPRoundTripTLS returns a [Filter] that attempts an HTTP round trip using the given [TLSConn].
func HTTPRoundTripTLS(ctx context.Context, rt Runtime, req *http.Request) Filter[TLSConn, HTTPResponse] {
	return startFilterService(func(conn TLSConn) (HTTPResponse, error) {
		dialer := netxlite.NewNullDialer()
		tlsDialer := netxlite.NewSingleUseTLSDialer(conn)
		options := []netxlite.HTTPTransportOption{ /* empty */ }
		txp := netxlite.NewHTTPTransportWithOptions(rt.Logger(), dialer, tlsDialer, options...)
		return httpRoundTrip(ctx, rt, txp, req, conn.Trace, "tcp", conn.RemoteAddr().String())
	})
}

// HTTPRoundTripQUIC returns a [Filter] that attempts an HTTP round trip using the given [QUICConn].
func HTTPRoundTripQUIC(ctx context.Context, rt Runtime, req *http.Request, config *tls.Config) Filter[QUICConn, HTTPResponse] {
	return startFilterService(func(conn QUICConn) (HTTPResponse, error) {
		dialer := netxlite.NewSingleUseQUICDialer(conn)
		txp := netxlite.NewHTTP3Transport(rt.Logger(), dialer, config)
		return httpRoundTrip(ctx, rt, txp, req, conn.Trace, "udp", conn.RemoteAddr().String())
	})
}

func httpRoundTrip(ctx context.Context, rt Runtime, txp model.HTTPTransport,
	req *http.Request, trace Trace, network, address string) (HTTPResponse, error) {
	// make sure the request uses the context and also make sure
	// we are not going to accidentally mutate the request
	req = req.Clone(ctx)

	// TODO: make sure we're going to save network events
	// by reading from the trace when we're leaving

	// TODO: make sure we log when the HTTP transaction
	// started and terminated

	// start operation logger
	ol := logx.NewOperationLogger(
		rt.Logger(),
		"[#%d] HTTPRequest %s with %s/%s host=%s",
		0, // TODO
		req.URL.String(),
		address,
		network,
		req.Host,
	)

	// perform the actual HTTP round trip
	resp, err := txp.RoundTrip(req)

	// handle the error case
	if err != nil {
		ol.Stop(err)
		// TODO: save the HTTP request inside the runtime
		return HTTPResponse{}, err
	}

	// TODO: create sampler for measuring throttling
	// and collect the samples

	// read a snapshot of the response body
	const maxbody = 1 << 19 // TODO(bassosimone): allow to configure this value?
	reader := io.LimitReader(resp.Body, maxbody)
	body, err := netxlite.ReadAllContext(ctx, reader)

	// handle the error case
	if err != nil {
		ol.Stop(err)
		// TODO: save HTTP request and response inside the runtime
		return HTTPResponse{}, err
	}

	// handle success
	ol.Stop(nil)
	return HTTPResponse{BodySnap: body, Resp: resp, Trace: trace}, nil
}
