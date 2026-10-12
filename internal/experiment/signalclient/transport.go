package signalclient

//
// TCP + TLS with a reproduced Signal ClientHello, then the client's application protocol:
// an HTTP/1.1 WebSocket upgrade, an RFC 8441 extended CONNECT over HTTP/2, or an HTTPS GET.
//

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 6455 defines Sec-WebSocket-Accept with SHA-1
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
)

// libsignal 0.105.0's HTTP/2 client settings, from a decrypted capture: SETTINGS ENABLE_PUSH 0,
// INITIAL_WINDOW_SIZE 2097152, MAX_FRAME_SIZE 16384, MAX_HEADER_LIST_SIZE 16384, then a
// connection WINDOW_UPDATE of 5177345. x/net sends its settings in the same order.
const (
	libsignalStreamWindow     = 2097152
	libsignalConnWindowUpdate = 5177345
	libsignalMaxFrameSize     = 16384
	libsignalMaxHeaderList    = 16384
)

// tlsResult is what dialTLS observed.
type tlsResult struct {
	conn      net.Conn
	alpn      string
	leafKey   string
	handshake *model.ArchivalTLSOrQUICHandshakeResult
}

// dialTLS connects to address and handshakes with the given hello, verifying the chain against
// roots for sni (nil roots: the chain is recorded but not verified). Events go to trace. It
// returns a nil *tlsResult only when the TCP connect failed.
func dialTLS(ctx context.Context, trace *measurexlite.Trace, logger model.Logger, zero time.Time,
	address, sni string, hello Hello, roots *x509.CertPool, tags ...string) (*tlsResult, error) {
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err // already wrapped by netxlite (operation "connect")
	}
	return handshakeTLS(ctx, trace, zero, conn, address, sni, hello, roots, tags...)
}

// handshakeTLS runs the TLS handshake of hello on conn (a TCP connection, or a tunnel whose
// address is recorded as given). It closes conn on failure.
func handshakeTLS(ctx context.Context, trace *measurexlite.Trace, zero time.Time, conn net.Conn,
	address, sni string, hello Hello, roots *x509.CertPool, tags ...string) (*tlsResult, error) {
	tlsFailure := func(err error) (*tlsResult, error) {
		_ = conn.Close()
		return &tlsResult{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.TLSHandshakeOperation,
			fmt.Errorf("signal_client: %s: %w", hello.Name, err))
	}
	var (
		tconn interface {
			net.Conn
			HandshakeContext(context.Context) error
		}
		connState func() tls.ConnectionState
	)
	if hello.stdlibTLS12 {
		c := tls.Client(conn, &tls.Config{ServerName: sni, RootCAs: roots, InsecureSkipVerify: roots == nil, // #nosec G402 -- recorded, not trusted
			MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12, NextProtos: hello.alpn})
		tconn, connState = c, c.ConnectionState
	} else {
		spec, err := hello.Spec()
		if err != nil {
			return tlsFailure(err)
		}
		config := &utls.Config{ServerName: sni, RootCAs: roots, InsecureSkipVerify: roots == nil} // #nosec G402 -- recorded, not trusted
		uconn := utls.UClient(conn, config, utls.HelloCustom)
		if err := uconn.ApplyPreset(spec); err != nil {
			return tlsFailure(err)
		}
		tconn, connState = uconn, func() tls.ConnectionState { return stdlibState(uconn.ConnectionState()) }
	}
	started := time.Since(zero)
	err := tconn.HandshakeContext(ctx)
	finished := time.Since(zero)
	if err != nil {
		err = netxlite.NewErrWrapper(netxlite.ClassifyTLSHandshakeError, netxlite.TLSHandshakeOperation, err)
	}
	state := connState()
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(trace.Index(), started, "tcp", address,
		&tls.Config{ServerName: sni, NextProtos: hello.ALPN(), InsecureSkipVerify: roots == nil}, // #nosec G402 -- archival only
		state, err, finished, tags...)
	if err != nil {
		_ = conn.Close()
		return &tlsResult{handshake: hs}, err
	}
	return &tlsResult{conn: tconn, alpn: state.NegotiatedProtocol, leafKey: leafKeyType(state), handshake: hs}, nil
}

// stdlibState converts uTLS's connection state for the archival helpers.
func stdlibState(s utls.ConnectionState) tls.ConnectionState {
	return tls.ConnectionState{
		Version:            s.Version,
		HandshakeComplete:  s.HandshakeComplete,
		CipherSuite:        s.CipherSuite,
		NegotiatedProtocol: s.NegotiatedProtocol,
		ServerName:         s.ServerName,
		PeerCertificates:   s.PeerCertificates,
	}
}

func leafKeyType(s tls.ConnectionState) string {
	if len(s.PeerCertificates) == 0 {
		return ""
	}
	return s.PeerCertificates[0].PublicKeyAlgorithm.String()
}

// upgradeResult is the server's answer to a WebSocket upgrade or extended CONNECT.
type upgradeResult struct {
	status          int
	signalTimestamp bool
	extensions      string // Sec-WebSocket-Extensions the server accepted
	ws              *wsConn
	closer          io.Closer
}

// errUpgradeRefused means the server answered the upgrade with something other than success.
var errUpgradeRefused = errors.New("signal_client: websocket upgrade refused")

// upgradeRefused records errUpgradeRefused with its failure string.
func upgradeRefused() error {
	return newFailure(FailureWebSocketUpgradeRefused, netxlite.HTTPRoundTripOperation, errUpgradeRefused)
}

// deflateAccepted is whether a Sec-WebSocket-Extensions answer accepts permessage-deflate.
func deflateAccepted(extensions string) bool {
	for _, ext := range strings.Split(extensions, ",") {
		name, _, _ := strings.Cut(ext, ";")
		if strings.EqualFold(strings.TrimSpace(name), "permessage-deflate") {
			return true
		}
	}
	return false
}

// errUpgradeInvalid means a 101 answer lacked the headers RFC 6455 section 4.1 requires.
var errUpgradeInvalid = errors.New("signal_client: invalid websocket upgrade response")

// websocketAccept is the Sec-WebSocket-Accept value a server must return for key.
func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")) // #nosec G401 -- RFC 6455 section 4.2.2
	return base64.StdEncoding.EncodeToString(sum[:])
}

// newWebSocketKey returns a random Sec-WebSocket-Key.
func newWebSocketKey() (string, error) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// websocketH1 performs Android OkHttp's HTTP/1.1 WebSocket upgrade on an established TLS conn.
func websocketH1(ctx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	wsKey, err := newWebSocketKey()
	if err != nil {
		return nil, err
	}
	// OkHttp 5.3.2's order: RealWebSocket.connect's headers, then the app's User-Agent
	// interceptor (Signal's StandardUserAgentInterceptor), then BridgeInterceptor's Host and
	// Accept-Encoding. All of it is inside TLS; only the encrypted record sizes are visible.
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + wsKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Extensions: permessage-deflate\r\n" +
		"User-Agent: " + userAgent + "\r\n" +
		"Host: " + host + "\r\n" +
		"Accept-Encoding: gzip\r\n\r\n"
	return upgradeH1(ctx, conn, req, wsKey)
}

// upgradeH1 sends an HTTP/1.1 WebSocket upgrade request and checks the answer.
func upgradeH1(ctx context.Context, conn net.Conn, req, wsKey string) (*upgradeResult, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		defer func() { _ = conn.SetDeadline(time.Time{}) }()
	}
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
	}
	out := &upgradeResult{status: resp.StatusCode, signalTimestamp: resp.Header.Get("X-Signal-Timestamp") != "",
		extensions: resp.Header.Get("Sec-WebSocket-Extensions")}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close() // first: Body.Close would otherwise drain an endless body until the deadline
		_ = resp.Body.Close()
		return out, upgradeRefused()
	}
	// as OkHttp and tungstenite do, accept the upgrade only with the headers RFC 6455 requires
	if !httpguts.HeaderValuesContainsToken(resp.Header["Upgrade"], "websocket") ||
		!httpguts.HeaderValuesContainsToken(resp.Header["Connection"], "upgrade") ||
		resp.Header.Get("Sec-WebSocket-Accept") != websocketAccept(wsKey) {
		_ = conn.Close()
		return out, newFailure(FailureWebSocketUpgradeInvalid, netxlite.HTTPRoundTripOperation, errUpgradeInvalid)
	}
	out.ws = newWSConn(br, conn)
	out.ws.deflate = deflateAccepted(out.extensions)
	out.closer = conn
	return out, nil
}

// websocketH2 performs libsignal's RFC 8441 extended CONNECT on an established TLS conn. The
// stream lives as long as ctx; waiting for the response headers is bounded by connectCtx.
func websocketH2(ctx, connectCtx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	t1 := &http.Transport{HTTP2: &http.HTTP2Config{
		MaxReceiveBufferPerStream:     libsignalStreamWindow,
		MaxReceiveBufferPerConnection: libsignalConnWindowUpdate,
		MaxReadFrameSize:              libsignalMaxFrameSize,
	}}
	//lint:ignore SA1019 x/net's HTTP/2 client is the one that runs on a uTLS conn: net/http's ClientConn dials itself and recognizes h2 only on *crypto/tls.Conn
	t2, err := http2.ConfigureTransports(t1)
	if err != nil {
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	//lint:ignore SA1019 x/net's HTTP/2 client is the one that runs on a uTLS conn: net/http's ClientConn dials itself and recognizes h2 only on *crypto/tls.Conn
	t2.MaxHeaderListSize = libsignalMaxHeaderList
	cconn := &causeConn{Conn: conn}
	//lint:ignore SA1019 x/net's HTTP/2 client is the one that runs on a uTLS conn: net/http's ClientConn dials itself and recognizes h2 only on *crypto/tls.Conn
	cc, err := t2.NewClientConn(cconn)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
	}
	pr, pw := io.Pipe()
	reqCtx, cancelReq := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodConnect, "https://"+host+path, pr)
	if err != nil {
		cancelReq()
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	req.Header[":protocol"] = []string{"websocket"}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Sec-WebSocket-Version", "13")
	stopHeaderTimer := context.AfterFunc(connectCtx, cancelReq) // no headers within the connect budget
	resp, err := cc.RoundTrip(req)
	headerTimeout := !stopHeaderTimer() && ctx.Err() == nil // the timer fired, not the parent ctx
	if headerTimeout {
		err = context.DeadlineExceeded
	}
	if err != nil {
		cancelReq()
		_ = cc.Close() // before closing any response body (it may wait for the write lock)
		_ = pw.Close()
		if resp != nil {
			_ = resp.Body.Close()
		}
		if strings.Contains(err.Error(), "extended connect not supported") { // x/net's error is unexported
			return nil, newFailure(FailureExtendedConnectNotSupported, netxlite.HTTPRoundTripOperation, err)
		}
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
	}
	out := &upgradeResult{status: resp.StatusCode, signalTimestamp: resp.Header.Get("X-Signal-Timestamp") != ""}
	if resp.StatusCode/100 != 2 { // libsignal accepts any 2xx
		cancelReq()
		_ = cc.Close() // first: closing the body may wait for the connection's write lock
		_ = pw.Close()
		_ = resp.Body.Close()
		return out, upgradeRefused()
	}
	out.ws = newWSConn(resp.Body, pw)
	out.ws.cause = cconn.cause
	out.closer = closerFunc(func() error {
		cancelReq()
		err := cc.Close() // first, for the same reason
		_ = pw.Close()
		_ = resp.Body.Close()
		return err
	})
	return out, nil
}

// causeConn remembers the first error the network returned on Read. When the peer closes or
// resets the connection, x/net closes its own socket in reaction, and later reads and writes see
// that local close ("use of closed network connection", "client conn is closed") instead of the
// peer's action. The recorded cause lets us report what happened on the wire.
type causeConn struct {
	net.Conn
	mu  sync.Mutex
	err error
}

func (c *causeConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if err != nil {
		c.mu.Lock()
		if c.err == nil {
			c.err = err
		}
		c.mu.Unlock()
	}
	return n, err
}

// cause returns the first read error from the network, or nil.
func (c *causeConn) cause() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

// isLocalCloseError reports errors that only say our side of the connection is closed.
func isLocalCloseError(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) ||
		strings.Contains(err.Error(), "http2: client conn is closed")
}

// classifyHTTP2 gives HTTP/2 stream and connection failures their failure strings: x/net reports
// a peer's RST_STREAM as a StreamError, and a connection that died under the client as the
// unexported "client conn is closed". It returns nil for other errors.
func classifyHTTP2(err error, operation string) error {
	var se http2.StreamError
	switch {
	case errors.As(err, &se):
		return newFailure(FailureHTTP2StreamReset, operation, err)
	case strings.Contains(err.Error(), "http2: client conn is closed"):
		return newFailure(netxlite.FailureEOFError, operation, err)
	}
	return nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// httpsGet sends one GET over an established TLS conn, using HTTP/2 when it was negotiated,
// as OkHttp does, and returns the status and whether Signal's timestamp header was present.
// It reads at most 64 KiB of the body; an error while reading it is returned with the status.
// It consumes conn: the conn is closed before returning.
func httpsGet(ctx context.Context, conn net.Conn, alpn, host, path, userAgent string) (int, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+path, nil)
	if err != nil {
		return 0, false, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")
	var resp *http.Response
	if alpn == "h2" {
		//lint:ignore SA1019 x/net's HTTP/2 client is the one that runs on a uTLS conn: net/http's ClientConn dials itself and recognizes h2 only on *crypto/tls.Conn
		cc, err := (&http2.Transport{}).NewClientConn(conn)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
		defer func() { _ = cc.Close() }()
		resp, err = cc.RoundTrip(req)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
	} else {
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		req.Header.Set("Connection", "close")
		if err := req.Write(conn); err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
		}
		resp, err = http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
		}
	}
	_, bodyErr := netxlite.CopyContext(ctx, io.Discard, io.LimitReader(resp.Body, 1<<16))
	_ = conn.Close() // first: Body.Close would otherwise drain an endless body until the deadline
	_ = resp.Body.Close()
	ts := resp.Header.Get("X-Signal-Timestamp") != ""
	if bodyErr != nil {
		return resp.StatusCode, ts, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, bodyErr)
	}
	return resp.StatusCode, ts, nil
}
