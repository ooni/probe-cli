package signalclient

//
// libsignal's reflector tunnel (rust/net/infra/src/tcp_ssl/proxy/reflector.rs): an HTTP/1.1
// WebSocket upgrade of GET /tls-tunnel through a domain front, naming the Signal host in
// X-Signal-Host; binary frames then carry the client's own TLS connection to that host.
//

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// maxTunnelFrame is libsignal's MAX_OUTBOUND_FRAME_SIZE: writes are split into binary frames of
// at most 64 KiB.
const maxTunnelFrame = 64 * 1024

// reflectorPath is the reflectors' tunnel endpoint (REFLECTOR_PROVIDERS_PROD).
const reflectorPath = "/tls-tunnel"

// tunnelUpgrade opens the reflector tunnel to target on an established TLS conn to the front.
// The request has tungstenite's headers, as libsignal sends them, plus X-Signal-Host. A
// User-Agent is added only when userAgent is not empty. No extension is offered, so the
// tunnel's frames are never compressed.
func tunnelUpgrade(ctx context.Context, conn net.Conn, hostHeader, target, userAgent string) (*upgradeResult, error) {
	wsKey, err := newWebSocketKey()
	if err != nil {
		return nil, err
	}
	req := "GET " + reflectorPath + " HTTP/1.1\r\n" +
		"Host: " + hostHeader + "\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: " + wsKey + "\r\n" +
		"X-Signal-Host: " + target + "\r\n"
	if userAgent != "" {
		req += "User-Agent: " + userAgent + "\r\n"
	}
	up, err := upgradeH1(ctx, conn, req+"\r\n", wsKey)
	if err == nil {
		up.ws.deflate = false // not offered: a compressed frame is a protocol violation
	}
	return up, err
}

// tunnelConn is a net.Conn over a reflector tunnel's binary frames.
type tunnelConn struct {
	net.Conn // the TLS connection to the front: addresses, deadlines and Close
	ws       *wsConn
	rmu      sync.Mutex
	pending  []byte
}

var _ net.Conn = &tunnelConn{}

// errTunnelText means the reflector sent a text frame, which libsignal treats as an error.
var errTunnelText = errors.New("signal_client: reflector tunnel received a text frame")

func newTunnelConn(up *upgradeResult, conn net.Conn) *tunnelConn {
	return &tunnelConn{Conn: conn, ws: up.ws}
}

// Read returns the payload of binary frames, answers pings, skips pongs, and reports a close
// frame as io.EOF.
func (c *tunnelConn) Read(p []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	for len(c.pending) == 0 {
		op, payload, err := c.ws.readFrame()
		if err != nil {
			return 0, readFailure(err)
		}
		switch op {
		case opBinary:
			c.pending = payload
		case opPing:
			_ = c.ws.writeFrame(opPong, payload)
		case opClose:
			return 0, eofFailure()
		case opPong:
		default:
			return 0, malformed(errTunnelText)
		}
	}
	n := copy(p, c.pending)
	c.pending = c.pending[n:]
	return n, nil
}

// Write sends p in binary frames of at most maxTunnelFrame bytes.
func (c *tunnelConn) Write(p []byte) (int, error) {
	for off := 0; off < len(p); off += maxTunnelFrame {
		end := min(off+maxTunnelFrame, len(p))
		if err := c.ws.writeFrame(opBinary, p[off:end]); err != nil {
			return off, err
		}
	}
	return len(p), nil
}

// SetDeadline and friends apply to the front connection, which carries every frame.
func (c *tunnelConn) SetDeadline(t time.Time) error      { return c.Conn.SetDeadline(t) }
func (c *tunnelConn) SetReadDeadline(t time.Time) error  { return c.Conn.SetReadDeadline(t) }
func (c *tunnelConn) SetWriteDeadline(t time.Time) error { return c.Conn.SetWriteDeadline(t) }
