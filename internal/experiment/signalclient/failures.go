package signalclient

//
// Failure strings and how errors become them. Errors are netxlite.ErrWrapper values, so that
// measurexlite records each with its failure string and the operation that failed.
//

import (
	"errors"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.org/x/net/http2"
)

// Experiment-specific failure strings (in addition to df-007's).
const (
	// FailureExtendedConnectNotSupported: the HTTP/2 server did not offer RFC 8441 extended CONNECT.
	FailureExtendedConnectNotSupported = "extended_connect_not_supported"

	// FailureWebSocketUpgradeRefused: the server answered the WebSocket upgrade or extended
	// CONNECT with a status other than success (see the target's status).
	FailureWebSocketUpgradeRefused = "websocket_upgrade_refused"

	// FailureWebSocketUpgradeInvalid: the server answered the HTTP/1.1 upgrade with 101 but without
	// the Upgrade, Connection or Sec-WebSocket-Accept headers RFC 6455 requires, which the apps'
	// WebSocket clients reject.
	FailureWebSocketUpgradeInvalid = "websocket_upgrade_invalid"

	// FailureUnexpectedMessage: the server sent a WebSocket message the flow does not allow.
	FailureUnexpectedMessage = "unexpected_message"

	// FailureMalformedMessage: a WebSocket message was not a valid WebSocketMessage protobuf, could
	// not be inflated, or exceeded the size limit, or a frame violated RFC 6455 or RFC 7692.
	FailureMalformedMessage = "malformed_message"

	// FailureHTTP2StreamReset: the peer reset the HTTP/2 stream carrying the WebSocket.
	FailureHTTP2StreamReset = "http2_stream_reset"
)

// newFailure wraps err so that it is recorded with the given failure string.
func newFailure(failure, operation string, err error) error {
	return &netxlite.ErrWrapper{Failure: failure, Operation: operation, WrappedErr: err}
}

// errNoAddress means DNS returned no usable address.
var errNoAddress = newFailure(netxlite.FailureDNSNoAnswer, netxlite.ResolveOperation, errors.New("no usable address"))

// errUnexpectedMessage means the server sent something the flow does not allow.
var errUnexpectedMessage = newFailure(FailureUnexpectedMessage, netxlite.ReadOperation, errors.New("signal_client: unexpected message"))

// malformed records an invalid message or frame with its failure string.
func malformed(err error) error {
	return newFailure(FailureMalformedMessage, netxlite.ReadOperation, err)
}

// errUpgradeRefused means the server answered the upgrade with something other than success.
var errUpgradeRefused = errors.New("signal_client: websocket upgrade refused")

// upgradeRefused records errUpgradeRefused with its failure string.
func upgradeRefused() error {
	return newFailure(FailureWebSocketUpgradeRefused, netxlite.HTTPRoundTripOperation, errUpgradeRefused)
}

// errUpgradeInvalid means a 101 answer lacked the headers RFC 6455 section 4.1 requires.
var errUpgradeInvalid = errors.New("signal_client: invalid websocket upgrade response")

// withCause replaces an error that only says our side is closed by the network error that caused
// the close, when ws knows it.
func withCause(ws *wsConn, err error) error {
	if ws.cause != nil && isLocalCloseError(err) {
		if cause := ws.cause(); cause != nil && !isLocalCloseError(cause) {
			return cause
		}
	}
	return err
}

// readFailure wraps an error from reading the WebSocket with its failure string.
func readFailure(err error) error {
	if errors.Is(err, errFrameTooLarge) || errors.Is(err, errInflate) || errors.Is(err, errProtocolViolation) {
		return malformed(err)
	}
	if h2err := classifyHTTP2(err, netxlite.ReadOperation); h2err != nil {
		return h2err
	}
	return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
}

// eofFailure is a WebSocket close from the server.
func eofFailure() error {
	return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, io.EOF)
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

// x/net's HTTP/2 client reports these conditions with unexported errors, so they are matched by
// text (golang.org/x/net/http2 v0.60.0; TestQANoExtendedConnect and TestQAStreamResetAfterAddress
// fail if the text changes).
const (
	xnetExtendedConnectNotSupported = "extended connect not supported"
	xnetClientConnClosed            = "http2: client conn is closed"
)

// isLocalCloseError reports errors that only say our side of the connection is closed.
func isLocalCloseError(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) ||
		strings.Contains(err.Error(), xnetClientConnClosed)
}

// classifyHTTP2 gives HTTP/2 stream and connection failures their failure strings: x/net reports
// a peer's RST_STREAM as a StreamError, and a connection that died under the client as the
// unexported "client conn is closed". It returns nil for other errors.
func classifyHTTP2(err error, operation string) error {
	var se http2.StreamError
	switch {
	case errors.As(err, &se):
		return newFailure(FailureHTTP2StreamReset, operation, err)
	case strings.Contains(err.Error(), xnetClientConnClosed):
		return newFailure(netxlite.FailureEOFError, operation, err)
	}
	return nil
}
