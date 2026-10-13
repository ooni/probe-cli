package signalclient

//
// libsignal's HTTP/2: its client settings, and the RFC 8441 extended CONNECT that carries its
// WebSockets. x/net's HTTP/2 client is used directly because it is the one that runs on a uTLS
// connection: net/http's dials by itself and recognizes h2 only on a *crypto/tls.Conn.
//
//lint:file-ignore SA1019 x/net's HTTP/2 client is the one that runs on a uTLS conn (see above)

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

// newH2ClientConn starts an HTTP/2 client connection over an established TLS conn, with t2's
// settings (nil: x/net's defaults).
func newH2ClientConn(conn net.Conn, t2 *http2.Transport) (*http2.ClientConn, error) {
	if t2 == nil {
		t2 = &http2.Transport{}
	}
	return t2.NewClientConn(conn)
}

// websocketH2 performs libsignal's RFC 8441 extended CONNECT on an established TLS conn. The
// stream lives as long as ctx; waiting for the response headers is bounded by connectCtx.
func websocketH2(ctx, connectCtx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	t1 := &http.Transport{HTTP2: &http.HTTP2Config{
		MaxReceiveBufferPerStream:     libsignalStreamWindow,
		MaxReceiveBufferPerConnection: libsignalConnWindowUpdate,
		MaxReadFrameSize:              libsignalMaxFrameSize,
	}}
	t2, err := http2.ConfigureTransports(t1)
	if err != nil {
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	t2.MaxHeaderListSize = libsignalMaxHeaderList
	cconn := &causeConn{Conn: conn}
	cc, err := newH2ClientConn(cconn, t2)
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
	var resp *http.Response
	teardown := func() error {
		cancelReq()
		err := cc.Close() // first: closing the response body may wait for the connection's write lock
		_ = pw.Close()
		if resp != nil {
			_ = resp.Body.Close()
		}
		return err
	}
	stopHeaderTimer := context.AfterFunc(connectCtx, cancelReq) // no headers within the operation's limit
	resp, err = cc.RoundTrip(req)
	// stopHeaderTimer returns false once the timer has run: the headers timed out, unless ctx ended
	if timerRan := !stopHeaderTimer(); timerRan && ctx.Err() == nil {
		err = context.DeadlineExceeded
	}
	if err != nil {
		_ = teardown()
		if strings.Contains(err.Error(), xnetExtendedConnectNotSupported) {
			return nil, newFailure(FailureExtendedConnectNotSupported, netxlite.HTTPRoundTripOperation, err)
		}
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
	}
	out := &upgradeResult{answer: newAnswer(resp)}
	if resp.StatusCode/100 != 2 { // libsignal accepts any 2xx
		out.response, _ = sampleResponse(ctx, resp) // bounded, within the operation's limit
		_ = teardown()
		return out, upgradeRefused()
	}
	out.ws = newWSConn(resp.Body, pw)
	out.ws.cause = cconn.cause
	out.closer = closerFunc(teardown)
	return out, nil
}
