package signalclient

//
// Signal's WebSocket protocol: RFC 6455 framing carrying WebSocketMessage protobufs
// (signalpb/signal.proto, after libsignal's chat_websocket.proto and chat_provisioning.proto).
//
// Hand-written rather than gorilla/websocket's: the WebSocket must also run over an RFC 8441
// HTTP/2 stream, and the HTTP/1.1 upgrade must send OkHttp's headers in OkHttp's order (both
// are visible to a censor that terminates TLS, and the order to anyone comparing with the app).
//

import (
	"bufio"
	"bytes"
	"compress/flate"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/experiment/signalclient/signalpb"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"google.golang.org/protobuf/proto"
)

// WebSocket opcodes (RFC 6455 section 5.2). Control frames are those from opClose up, which
// readRawFrame relies on.
const (
	opContinuation = 0x0
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// The frame header (RFC 6455 section 5.2): the first byte's flags and opcode, the second byte's
// mask flag and payload length, which 126 and 127 extend to 16 and 64 bits.
const (
	finBit       = 0x80
	rsv1Bit      = 0x40 // permessage-deflate's "compressed" flag (RFC 7692)
	rsv2rsv3Bits = 0x30
	opcodeBits   = 0x0F
	maskBit      = 0x80
	lengthBits   = 0x7F
	length16     = 126
	length64     = 127

	maxFrameHeader    = 2 + 8 + 4 // with a 64-bit length and a mask
	maxControlPayload = 125
)

// maxFramePayload bounds what we accept from the server (Signal's messages here are tiny).
const maxFramePayload = 1 << 20

// permessage-deflate (RFC 7692 section 7.2.2): the sender strips the 0x00 0x00 0xff 0xff that
// ends each compressed message; the receiver appends it, and an empty final block, to inflate the
// message. The sender's LZ77 window, and so the dictionary of the next message, is 32 KiB.
var deflateTail = []byte{0x00, 0x00, 0xff, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff}

const deflateWindow = 1 << 15

// errFrameTooLarge means the server sent a frame larger than maxFramePayload.
var errFrameTooLarge = errors.New("signal_client: websocket frame too large")

// errInflate means a permessage-deflate message could not be decompressed.
var errInflate = errors.New("signal_client: cannot inflate websocket message")

// errProtocolViolation means the server sent a frame RFC 6455 or RFC 7692 forbids, which the
// apps' WebSocket clients reject: a reserved opcode or bit, an oversized or fragmented control
// frame, a compressed frame where compression is not allowed, or fragments out of order.
var errProtocolViolation = errors.New("signal_client: websocket protocol violation")

// wsConn is a minimal client-side WebSocket over any byte stream (an HTTP/1.1 upgraded
// connection or an RFC 8441 HTTP/2 stream). Client frames are masked, as RFC 6455 and 8441 require.
type wsConn struct {
	r  *bufio.Reader
	w  io.Writer
	mu sync.Mutex

	// deflate is whether permessage-deflate was negotiated: only then may messages be compressed
	deflate bool

	// a message whose final fragment has not arrived yet (control frames may come in between)
	fragmenting       bool
	partial           []byte
	partialOp         byte
	partialCompressed bool

	// cause, when set, returns the first error the network returned under this WebSocket (see
	// causeConn): it explains local-close errors seen by later reads and writes.
	cause func() error

	// window holds the last 32 KiB the server sent compressed (RFC 7692 permessage-deflate
	// with context takeover): each message is inflated with it as the dictionary.
	window []byte
}

// inflate decompresses one permessage-deflate message.
func (c *wsConn) inflate(p []byte) ([]byte, error) {
	r := flate.NewReaderDict(io.MultiReader(bytes.NewReader(p), bytes.NewReader(deflateTail)), c.window)
	out, err := netxlite.ReadAllContext(context.Background(), io.LimitReader(r, maxFramePayload+1)) // in memory
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInflate, err)
	}
	if len(out) > maxFramePayload {
		return nil, errFrameTooLarge
	}
	c.window = append(c.window, out...)
	if len(c.window) > deflateWindow {
		c.window = c.window[len(c.window)-deflateWindow:]
	}
	return out, nil
}

func newWSConn(r io.Reader, w io.Writer) *wsConn {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &wsConn{r: br, w: w}
}

// writeFrame sends one final, masked frame.
func (c *wsConn) writeFrame(opcode byte, payload []byte) error {
	var hdr [maxFrameHeader]byte
	hdr[0] = finBit | opcode
	n := len(payload)
	off := 2
	switch {
	case n < length16:
		hdr[1] = maskBit | byte(n)
	case n <= math.MaxUint16:
		hdr[1] = maskBit | length16
		binary.BigEndian.PutUint16(hdr[2:], uint16(n))
		off = 4
	default:
		hdr[1] = maskBit | length64
		binary.BigEndian.PutUint64(hdr[2:], uint64(n))
		off = 10
	}
	mask := hdr[off : off+4]
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	frame := make([]byte, 0, off+4+n)
	frame = append(frame, hdr[:off+4]...)
	for i, b := range payload {
		frame = append(frame, b^mask[i%4])
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.w.Write(frame)
	return err
}

// readFrame returns the next complete message (continuation frames joined, inflated when
// compressed) or control frame. Control frames that arrive between the fragments of a message
// are returned as they come, and assembly of the message continues on the next call.
func (c *wsConn) readFrame() (byte, []byte, error) {
	for {
		f, err := c.readRawFrame()
		if err != nil {
			return 0, nil, err
		}
		if f.op >= opClose { // control frames are never fragmented
			return f.op, f.data, nil
		}
		if f.op != opContinuation { // first fragment of a new message
			c.partial, c.partialOp, c.partialCompressed = nil, f.op, f.rsv1
		}
		c.partial = append(c.partial, f.data...)
		if !f.fin {
			c.fragmenting = true
			continue
		}
		op, payload, compressed := c.partialOp, c.partial, c.partialCompressed
		c.partial, c.fragmenting = nil, false
		if compressed {
			out, err := c.inflate(payload)
			return op, out, err
		}
		return op, payload, nil
	}
}

// rawFrame is one frame as read from the wire, unmasked.
type rawFrame struct {
	fin, rsv1 bool
	op        byte
	data      []byte
}

// readRawFrame reads one frame, rejecting what RFC 6455 and RFC 7692 forbid given the message
// being assembled.
func (c *wsConn) readRawFrame() (rawFrame, error) {
	var h [2]byte
	if _, err := io.ReadFull(c.r, h[:]); err != nil {
		return rawFrame{}, err
	}
	f := rawFrame{fin: h[0]&finBit != 0, rsv1: h[0]&rsv1Bit != 0, op: h[0] & opcodeBits}
	masked, n := h[1]&maskBit != 0, uint64(h[1]&lengthBits)
	switch op := f.op; {
	case h[0]&rsv2rsv3Bits != 0: // no extension we negotiate defines them
		return f, fmt.Errorf("%w: RSV2 or RSV3 set", errProtocolViolation)
	case op > opBinary && op < opClose, op > opPong:
		return f, fmt.Errorf("%w: reserved opcode %#x", errProtocolViolation, op)
	case op >= opClose && (!f.fin || n > maxControlPayload):
		return f, fmt.Errorf("%w: fragmented or oversized control frame", errProtocolViolation)
	case f.rsv1 && (!c.deflate || op == opContinuation || op >= opClose): // RFC 7692 section 6
		return f, fmt.Errorf("%w: RSV1 set where compression is not allowed", errProtocolViolation)
	case op == opContinuation && !c.fragmenting:
		return f, fmt.Errorf("%w: continuation frame without a message", errProtocolViolation)
	case op != opContinuation && op < opClose && c.fragmenting:
		return f, fmt.Errorf("%w: new message before the previous one ended", errProtocolViolation)
	}
	switch n { // extended payload lengths
	case length16:
		var b [2]byte
		if _, err := io.ReadFull(c.r, b[:]); err != nil {
			return f, err
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case length64:
		var b [8]byte
		if _, err := io.ReadFull(c.r, b[:]); err != nil {
			return f, err
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > maxFramePayload || (f.op < opClose && n+uint64(len(c.partial)) > maxFramePayload) {
		return f, errFrameTooLarge
	}
	var mask [4]byte
	if masked { // servers must not mask, but tolerate it
		if _, err := io.ReadFull(c.r, mask[:]); err != nil {
			return f, err
		}
	}
	f.data = make([]byte, n)
	if _, err := io.ReadFull(c.r, f.data); err != nil {
		return f, err
	}
	if masked {
		for i := range f.data {
			f.data[i] ^= mask[i%4]
		}
	}
	return f, nil
}

// requestMessage encodes a WebSocketMessage carrying req.
func requestMessage(req *signalpb.WebSocketRequestMessage) []byte {
	return marshal(&signalpb.WebSocketMessage{Type: signalpb.WebSocketMessage_REQUEST.Enum(), Request: req})
}

// responseMessage encodes a WebSocketMessage carrying resp.
func responseMessage(resp *signalpb.WebSocketResponseMessage) []byte {
	return marshal(&signalpb.WebSocketMessage{Type: signalpb.WebSocketMessage_RESPONSE.Enum(), Response: resp})
}

func marshal(msg proto.Message) []byte {
	b, err := proto.Marshal(msg)
	runtimex.PanicOnError(err, "signal_client: cannot marshal a protobuf") // proto2, no required fields
	return b
}

// errMalformedProto means a protobuf could not be parsed.
var errMalformedProto = errors.New("signal_client: malformed protobuf")

// parseMessage decodes a WebSocketMessage, which must carry a request or a response.
func parseMessage(b []byte) (*signalpb.WebSocketMessage, error) {
	msg := &signalpb.WebSocketMessage{}
	if err := proto.Unmarshal(b, msg); err != nil {
		return nil, fmt.Errorf("%w: %w", errMalformedProto, err)
	}
	if msg.Request == nil && msg.Response == nil {
		return nil, fmt.Errorf("%w: neither request nor response", errMalformedProto)
	}
	return msg, nil
}

// errNoProvisioningAddress means a ProvisioningAddress had no address field.
var errNoProvisioningAddress = errors.New("signal_client: ProvisioningAddress without an address")

// provisioningAddress decodes a ProvisioningAddress and returns its address, which must be present.
func provisioningAddress(body []byte) (string, error) {
	pa := &signalpb.ProvisioningAddress{}
	if err := proto.Unmarshal(body, pa); err != nil {
		return "", fmt.Errorf("%w: %w", errMalformedProto, err)
	}
	if pa.Address == nil {
		return "", errNoProvisioningAddress
	}
	return pa.GetAddress(), nil
}
