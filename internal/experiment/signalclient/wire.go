package signalclient

//
// Signal's WebSocket protocol: RFC 6455 framing carrying WebSocketMessage protobufs
// (libsignal rust/net/src/proto/chat_websocket.proto, chat_provisioning.proto).
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

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"google.golang.org/protobuf/encoding/protowire"
)

// WebSocket opcodes.
const (
	opContinuation = 0x0
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// maxFramePayload bounds what we accept from the server (Signal's messages here are tiny).
const maxFramePayload = 1 << 20

// errFrameTooLarge means the server sent a frame larger than maxFramePayload.
var errFrameTooLarge = errors.New("signal_client: websocket frame too large")

// wsConn is a minimal client-side WebSocket over any byte stream (an HTTP/1.1 upgraded
// connection or an RFC 8441 HTTP/2 stream). Client frames are masked, as RFC 6455 and 8441 require.
type wsConn struct {
	r  *bufio.Reader
	w  io.Writer
	mu sync.Mutex

	// a message whose final fragment has not arrived yet (control frames may come in between)
	partial           []byte
	partialOp         byte
	partialCompressed bool

	// window holds the last 32 KiB the server sent compressed (RFC 7692 permessage-deflate
	// with context takeover): each message is inflated with it as the dictionary.
	window []byte
}

// inflate decompresses one permessage-deflate message.
func (c *wsConn) inflate(p []byte) ([]byte, error) {
	tail := []byte{0x00, 0x00, 0xff, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff} // sync flush + final empty block
	r := flate.NewReaderDict(io.MultiReader(bytes.NewReader(p), bytes.NewReader(tail)), c.window)
	out, err := netxlite.ReadAllContext(context.Background(), io.LimitReader(r, maxFramePayload+1)) // in memory
	if err != nil {
		return nil, fmt.Errorf("signal_client: inflating websocket message: %w", err)
	}
	if len(out) > maxFramePayload {
		return nil, errFrameTooLarge
	}
	c.window = append(c.window, out...)
	if len(c.window) > 32768 {
		c.window = c.window[len(c.window)-32768:]
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
	var hdr [14]byte
	hdr[0] = 0x80 | opcode
	n := len(payload)
	off := 2
	switch {
	case n < 126:
		hdr[1] = 0x80 | byte(n)
	case n < 65536:
		hdr[1] = 0x80 | 126
		binary.BigEndian.PutUint16(hdr[2:], uint16(n))
		off = 4
	default:
		hdr[1] = 0x80 | 127
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

// readFrame returns the next complete message (continuation frames joined) or control frame.
// Control frames that arrive between the fragments of a message are returned as they come,
// and assembly of the message continues on the next call.
func (c *wsConn) readFrame() (byte, []byte, error) {
	for {
		var h [2]byte
		if _, err := io.ReadFull(c.r, h[:]); err != nil {
			return 0, nil, err
		}
		fin, op, masked := h[0]&0x80 != 0, h[0]&0x0F, h[1]&0x80 != 0
		n := uint64(h[1] & 0x7F)
		switch n {
		case 126:
			var b [2]byte
			if _, err := io.ReadFull(c.r, b[:]); err != nil {
				return 0, nil, err
			}
			n = uint64(binary.BigEndian.Uint16(b[:]))
		case 127:
			var b [8]byte
			if _, err := io.ReadFull(c.r, b[:]); err != nil {
				return 0, nil, err
			}
			n = binary.BigEndian.Uint64(b[:])
		}
		if n > maxFramePayload || (op < opClose && n+uint64(len(c.partial)) > maxFramePayload) {
			return 0, nil, errFrameTooLarge
		}
		var mask [4]byte
		if masked { // servers must not mask, but tolerate it
			if _, err := io.ReadFull(c.r, mask[:]); err != nil {
				return 0, nil, err
			}
		}
		data := make([]byte, n)
		if _, err := io.ReadFull(c.r, data); err != nil {
			return 0, nil, err
		}
		if masked {
			for i := range data {
				data[i] ^= mask[i%4]
			}
		}
		if op >= opClose { // control frames are never fragmented
			return op, data, nil
		}
		if op != opContinuation { // first fragment of a new message
			c.partial, c.partialOp, c.partialCompressed = nil, op, h[0]&0x40 != 0
		}
		c.partial = append(c.partial, data...)
		if !fin {
			continue
		}
		op, payload, compressed := c.partialOp, c.partial, c.partialCompressed
		c.partial = nil
		if compressed {
			out, err := c.inflate(payload)
			return op, out, err
		}
		return op, payload, nil
	}
}

// wsMessage is a decoded WebSocketMessage.
type wsMessage struct {
	Type     uint64 // 1 = REQUEST, 2 = RESPONSE
	Request  *wsRequest
	Response *wsResponse
}

// wsRequest is a WebSocketRequestMessage.
type wsRequest struct {
	Verb    string
	Path    string
	Body    []byte
	Headers []string
	ID      uint64
}

// wsResponse is a WebSocketResponseMessage.
type wsResponse struct {
	ID      uint64
	Status  uint32
	Message string
	Headers []string
	Body    []byte
}

const (
	msgTypeRequest  = 1
	msgTypeResponse = 2
)

func (r *wsRequest) marshal() []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, r.Verb)
	b = protowire.AppendTag(b, 2, protowire.BytesType)
	b = protowire.AppendString(b, r.Path)
	if r.Body != nil {
		b = protowire.AppendTag(b, 3, protowire.BytesType)
		b = protowire.AppendBytes(b, r.Body)
	}
	b = protowire.AppendTag(b, 4, protowire.VarintType)
	b = protowire.AppendVarint(b, r.ID)
	for _, h := range r.Headers {
		b = protowire.AppendTag(b, 5, protowire.BytesType)
		b = protowire.AppendString(b, h)
	}
	return wrapMessage(msgTypeRequest, 2, b)
}

func (r *wsResponse) marshal() []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, r.ID)
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, uint64(r.Status))
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendString(b, r.Message)
	for _, h := range r.Headers {
		b = protowire.AppendTag(b, 5, protowire.BytesType)
		b = protowire.AppendString(b, h)
	}
	if r.Body != nil {
		b = protowire.AppendTag(b, 4, protowire.BytesType)
		b = protowire.AppendBytes(b, r.Body)
	}
	return wrapMessage(msgTypeResponse, 3, b)
}

func wrapMessage(typ uint64, field protowire.Number, inner []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.VarintType)
	b = protowire.AppendVarint(b, typ)
	b = protowire.AppendTag(b, field, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)
	return b
}

// errMalformedProto means a protobuf could not be parsed.
var errMalformedProto = errors.New("signal_client: malformed protobuf")

// fields walks a protobuf message, calling fn for each varint or length-delimited field.
func fields(b []byte, fn func(num protowire.Number, v uint64, bytes []byte)) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return errMalformedProto
		}
		b = b[n:]
		switch typ {
		case protowire.VarintType:
			v, m := protowire.ConsumeVarint(b)
			if m < 0 {
				return errMalformedProto
			}
			fn(num, v, nil)
			b = b[m:]
		case protowire.BytesType:
			v, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return errMalformedProto
			}
			fn(num, 0, v)
			b = b[m:]
		default:
			m := protowire.ConsumeFieldValue(num, typ, b)
			if m < 0 {
				return errMalformedProto
			}
			b = b[m:]
		}
	}
	return nil
}

func parseMessage(b []byte) (*wsMessage, error) {
	msg := &wsMessage{}
	var reqBytes, respBytes []byte
	err := fields(b, func(num protowire.Number, v uint64, bs []byte) {
		switch num {
		case 1:
			msg.Type = v
		case 2:
			reqBytes = bs
		case 3:
			respBytes = bs
		}
	})
	if err != nil {
		return nil, err
	}
	if reqBytes != nil {
		r := &wsRequest{}
		if err := fields(reqBytes, func(num protowire.Number, v uint64, bs []byte) {
			switch num {
			case 1:
				r.Verb = string(bs)
			case 2:
				r.Path = string(bs)
			case 3:
				r.Body = bs
			case 4:
				r.ID = v
			case 5:
				r.Headers = append(r.Headers, string(bs))
			}
		}); err != nil {
			return nil, err
		}
		msg.Request = r
	}
	if respBytes != nil {
		r := &wsResponse{}
		if err := fields(respBytes, func(num protowire.Number, v uint64, bs []byte) {
			switch num {
			case 1:
				r.ID = v
			case 2:
				r.Status = uint32(min(v, math.MaxUint32)) // #nosec G115 -- clamped; HTTP statuses are tiny
			case 3:
				r.Message = string(bs)
			case 4:
				r.Body = bs
			case 5:
				r.Headers = append(r.Headers, string(bs))
			}
		}); err != nil {
			return nil, err
		}
		msg.Response = r
	}
	if msg.Request == nil && msg.Response == nil {
		return nil, fmt.Errorf("%w: neither request nor response", errMalformedProto)
	}
	return msg, nil
}

// provisioningAddress extracts ProvisioningAddress.address (field 1).
func provisioningAddress(body []byte) (string, error) {
	var addr string
	err := fields(body, func(num protowire.Number, _ uint64, bs []byte) {
		if num == 1 {
			addr = string(bs)
		}
	})
	return addr, err
}
