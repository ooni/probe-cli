package netplumbing

// This file contains the implementation of OOHTTPTransport.

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/idna"
)

// OOHTTPTransport is a reimplementation of http.Transport.
type OOHTTPTransport struct {
	// Transport is the underlying Transport, which we use for
	// establishing new possibly-encrypted connections.
	Transport *Transport

	// activeConns contains the active connections.
	activeConns map[string]httpxconn

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// httpxconn is either an http2con or an http1conn.
type httpxconn interface {
	// RoundTrip performs the HTTP round trip.
	RoundTrip(req *http.Request) (*http.Response, error)

	// Close closes the connection.
	Close() error
}

// RoundTrip performs the HTTP round trip.
func (txp *OOHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Scheme {
	case "http", "https":
	default:
		return nil, errors.New("netplumbing: unsupported scheme")
	}
	endpoint := txp.endpoint(req.URL.Host, txp.defaultPort(req.URL.Scheme))
	if conn := txp.popconn(endpoint); conn != nil {
		resp, err := conn.RoundTrip(req)
		if err == nil {
			txp.putconn(endpoint, conn)
			return resp, nil
		}
		conn.Close()
		if !errors.Is(err, errConnIsClosed) {
			return nil, err
		}
		// fallthrough
	}
	conn, err := txp.dial(req.Context(), req.URL.Scheme, endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := conn.RoundTrip(req)
	if err != nil {
		conn.Close()
		return nil, err
	}
	txp.putconn(endpoint, conn)
	return resp, nil
}

// defaultPort returns the default port for the given scheme.
func (txp *OOHTTPTransport) defaultPort(scheme string) string {
	switch scheme {
	case "https":
		return "443"
	default:
		return "80"
	}
}

// dial dials a new httpxconn
func (txp *OOHTTPTransport) dial(
	ctx context.Context, scheme, endpoint string) (httpxconn, error) {
	switch scheme {
	case "http":
		conn, err := txp.Transport.DialContext(ctx, "tcp", endpoint)
		if err != nil {
			return nil, err
		}
		return newhttp1conn(conn, txp.idleConnTimeout()), nil
	case "https":
		conn, state, err := txp.Transport.dialTLSContext(ctx, "tcp", endpoint)
		if err != nil {
			return nil, err
		}
		switch state.NegotiatedProtocol {
		case "h2":
			return newhttp2conn(conn, txp.idleConnTimeout())
		default:
			return newhttp1conn(conn, txp.idleConnTimeout()), nil
		}
	default:
		return nil, errors.New("netplumbing: unsupported scheme")
	}
}

// idleConnTimeout returns the idle conn timeout to use.
func (txp *OOHTTPTransport) idleConnTimeout() time.Duration {
	return 30 * time.Second
}

// endpoint constructs an endpoint to connect to from the
// value contained inside of the URL.Host field.
//
// License: 3-clause BSD + patent grant.
//
// Adapted from: golang.org/x/net/http2/transport.go
func (txp *OOHTTPTransport) endpoint(address, defaultPort string) string {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host, port = address, defaultPort
	}
	if conv, err := idna.ToASCII(host); err == nil {
		host = conv
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host + ":" + port
	}
	return net.JoinHostPort(host, port)
}

// popconn extracts the persistent conn from the cache and returns
// it, if available, otherwise it just returns nil.
func (txp *OOHTTPTransport) popconn(host string) httpxconn {
	defer txp.mu.Unlock()
	txp.mu.Lock()
	if conn, good := txp.activeConns[host]; good {
		delete(txp.activeConns, host)
		return conn
	}
	return nil
}

// putconn puts the connection back into the cache.
func (txp *OOHTTPTransport) putconn(host string, conn httpxconn) {
	defer txp.mu.Unlock()
	txp.mu.Lock()
	if txp.activeConns == nil {
		txp.activeConns = make(map[string]httpxconn)
	}
	if oldconn, good := txp.activeConns[host]; good {
		oldconn.Close() // give higher priority to new connection
	}
	txp.activeConns[host] = conn
}

// CloseIdleConnections closes idle connections.
func (txp *OOHTTPTransport) CloseIdleConnections() {
	defer txp.mu.Unlock()
	txp.mu.Lock()
	for _, conn := range txp.activeConns {
		conn.Close()
	}
	txp.activeConns = nil
}

// newhttp2conn creates a new http2 connection.
func newhttp2conn(conn net.Conn, idleConnTimeout time.Duration) (*http2conn, error) {
	txp := &http2.Transport{ReadIdleTimeout: idleConnTimeout}
	cc, err := txp.NewClientConn(conn)
	if err != nil {
		return nil, err
	}
	return &http2conn{cc: cc}, nil
}

// http2conn is a connection using http2. You should create a new
// instance of this struct using the nehttp2conn factory.
type http2conn struct {
	cc *http2.ClientConn
}

// RoundTrip performs the http round trip.
func (c *http2conn) RoundTrip(req *http.Request) (*http.Response, error) {
	if !c.cc.CanTakeNewRequest() {
		return nil, errConnIsClosed
	}
	return c.cc.RoundTrip(req)
}

// errConnIsClosed indicates that this connection has been closed.
var errConnIsClosed = errors.New("netplumbing: this connection has been closed")

// Close closes the underlying http2 connection.
func (c *http2conn) Close() error {
	// the underlying connection should have once semantics.
	return c.cc.Close()
}

// newhttp1conn creates a new http/1.1 connection.
func newhttp1conn(conn net.Conn, idleConnTimeout time.Duration) *http1conn {
	out := &http1conn{
		closedch:        make(chan interface{}),
		conn:            conn,
		idleConnTimeout: idleConnTimeout,
		pendingch:       make(chan *http1pendingreq),
		r:               bufio.NewReader(conn),
	}
	go out.readloop()
	return out
}

// http1conn is a connection using http/1.1. You should instantiate this
// struct using the newhttp1conn factory function.
type http1conn struct {
	// closedch is closed when the reader terminates.
	closedch chan interface{}

	// conn is the underlying conn.
	conn net.Conn

	// idleConnTime is the idle timeout for a connection.
	idleConnTimeout time.Duration

	// pendingch is where to posts pending requests.
	pendingch chan *http1pendingreq

	// r is the conn reader.
	r *bufio.Reader
}

// RoundTrip performs the round trip using the http/1.1 connection.
func (c *http1conn) RoundTrip(req *http.Request) (*http.Response, error) {
	respch := make(chan *http.Response)
	errch := make(chan error, 1)
	go func() {
		resp, err := c.roundTrip(req)
		if err != nil {
			errch <- err
			return
		}
		select {
		case respch <- resp:
		default:
			resp.Body.Close()
		}
	}()
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case resp := <-respch:
		return resp, nil
	case err := <-errch:
		return nil, err
	}
}

// roundTrip is the internal implementation of the round trip. We don't need
// to think about the context here, since we do that above.
func (c *http1conn) roundTrip(req *http.Request) (*http.Response, error) {
	// check whether we have a good conn.
	select {
	case <-c.closedch:
		return nil, errConnIsClosed
	default:
	}
	// we're good to send the request.
	if err := req.Write(c.conn); err != nil {
		return nil, err
	}
	// tell the reader we've got a request and also check
	// whether it exited just after we sent. If the reader
	// is servicing a previous request, we will block in
	// this spot until such a request is complete.
	pr := &http1pendingreq{
		errch:  make(chan error),
		req:    req,
		respch: make(chan *http.Response),
	}
	select {
	case c.pendingch <- pr:
	case <-c.closedch:
		return nil, errConnIsClosed
	}
	// if the reader accepted the request, then it's gonna
	// produce either a response or an error.
	select {
	case err := <-pr.errch:
		return nil, err
	case resp := <-pr.respch:
		return resp, nil
	}
}

// http1pendingreq is a pending http1 request.
type http1pendingreq struct {
	// errch is the error chan.
	errch chan error

	// req contains the pending request.
	req *http.Request

	// respch is the response chan.
	respch chan *http.Response
}

// readloop reads incoming requests.
func (c *http1conn) readloop() {
	defer close(c.closedch) // we've exited!
	for {
		c.conn.SetReadDeadline(time.Now().Add(c.idleConnTimeout))
		resp, err := http.ReadResponse(c.r, nil)
		select {
		case pr := <-c.pendingch:
			// we know which request this response belongs to
			// thus forward the error or the body
			if err != nil {
				pr.errch <- err
				return
			}
			c.conn.SetReadDeadline(time.Time{})
			resp.Request = pr.req
			reader, writer := io.Pipe()
			body := resp.Body
			resp.Body = reader
			pr.respch <- resp
			_, err := io.Copy(writer, body)
			writer.CloseWithError(err) // nil translates to EOF on the pipe
			body.Close()
			if err != nil || resp.Close {
				// If there is a read error or the client closed the
				// body before it finished reading it, then we cannot
				// continue using this connection. Note that the
				// http.Client code will read up to 2<<10 bytes of
				// the response body for us when redirecting.
				//
				// Likewise if we know that this connection is not
				// being used again, then stop using it.
				return
			}
		default:
			// received response with no pending request, so just
			// stop reading because what else to do?
			return
		}
	}
}

// Close closes the underlying connection.
func (c *http1conn) Close() error {
	// the underlying connection should have close semantics.
	return c.conn.Close()
}
