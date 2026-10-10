package signalclient

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"io"
	"testing"
)

func TestRequestAndResponseRoundTrip(t *testing.T) {
	req := &wsRequest{Verb: "GET", Path: "/v1/keepalive", Body: []byte{1, 2, 3}, ID: 1791563494213, Headers: []string{"a:b"}}
	msg, err := parseMessage(req.marshal())
	if err != nil || msg.Type != msgTypeRequest || msg.Request == nil {
		t.Fatalf("parse: %v %+v", err, msg)
	}
	if got := msg.Request; got.Verb != req.Verb || got.Path != req.Path || got.ID != req.ID || !bytes.Equal(got.Body, req.Body) || got.Headers[0] != "a:b" {
		t.Fatalf("request mismatch: %+v", got)
	}
	resp := &wsResponse{ID: 42, Status: 200, Message: "OK", Headers: []string{"X-Signal-Timestamp:1"}}
	msg, err = parseMessage(resp.marshal())
	if err != nil || msg.Type != msgTypeResponse || msg.Response.ID != 42 || msg.Response.Status != 200 || msg.Response.Message != "OK" {
		t.Fatalf("response: %v %+v", err, msg)
	}
}

func TestParseMessageRejectsGarbage(t *testing.T) {
	for _, b := range [][]byte{{0xff}, {0x08, 0x01}, {}} {
		if _, err := parseMessage(b); err == nil {
			t.Errorf("%x: expected an error", b)
		}
	}
}

func TestProvisioningAddress(t *testing.T) {
	body := append([]byte{0x0a, 24}, []byte("abcdefghijklmnopqrstuvwx")...)
	addr, err := provisioningAddress(body)
	if err != nil || addr != "abcdefghijklmnopqrstuvwx" {
		t.Fatalf("got %q %v", addr, err)
	}
}

// serverFrame builds an unmasked server frame.
func serverFrame(fin bool, rsv1 bool, op byte, payload []byte) []byte {
	b0 := op
	if fin {
		b0 |= 0x80
	}
	if rsv1 {
		b0 |= 0x40
	}
	out := []byte{b0}
	switch n := len(payload); {
	case n < 126:
		out = append(out, byte(n))
	case n < 65536:
		out = append(out, 126, byte(n>>8), byte(n))
	default:
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], uint64(n))
		out = append(append(out, 127), l[:]...)
	}
	return append(out, payload...)
}

func TestReadFrameLengthsAndFragments(t *testing.T) {
	small, mid, big := bytes.Repeat([]byte{1}, 10), bytes.Repeat([]byte{2}, 300), bytes.Repeat([]byte{3}, 70000)
	var stream bytes.Buffer
	stream.Write(serverFrame(true, false, opBinary, small))
	stream.Write(serverFrame(true, false, opBinary, mid))
	stream.Write(serverFrame(true, false, opBinary, big))
	stream.Write(serverFrame(false, false, opBinary, []byte("ab")))
	stream.Write(serverFrame(true, false, opPing, []byte("p"))) // control frame between fragments
	stream.Write(serverFrame(true, false, opContinuation, []byte("cd")))
	ws := newWSConn(&stream, io.Discard)
	for _, want := range [][]byte{small, mid, big, []byte("p"), []byte("abcd")} {
		_, got, err := ws.readFrame()
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("got %d bytes, err %v; want %d bytes", len(got), err, len(want))
		}
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	var stream bytes.Buffer
	var l [8]byte
	binary.BigEndian.PutUint64(l[:], maxFramePayload+1)
	stream.Write(append([]byte{0x82, 127}, l[:]...))
	if _, _, err := newWSConn(&stream, io.Discard).readFrame(); err != errFrameTooLarge {
		t.Fatalf("got %v", err)
	}
}

func TestWriteFrameIsMasked(t *testing.T) {
	var out bytes.Buffer
	ws := newWSConn(bytes.NewReader(nil), &out)
	payload := bytes.Repeat([]byte("x"), 200)
	if err := ws.writeFrame(opBinary, payload); err != nil {
		t.Fatal(err)
	}
	b := out.Bytes()
	if b[0] != 0x82 || b[1] != 0x80|126 || int(binary.BigEndian.Uint16(b[2:4])) != 200 {
		t.Fatalf("bad header % x", b[:4])
	}
	mask, data := b[4:8], b[8:]
	for i := range data {
		data[i] ^= mask[i%4]
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("payload does not unmask")
	}
}

// deflateMessages compresses messages with one shared context, as a server using
// permessage-deflate with context takeover does, and strips each sync-flush tail.
func deflateMessages(t *testing.T, msgs ...[]byte) [][]byte {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.BestCompression)
	var out [][]byte
	for _, m := range msgs {
		buf.Reset()
		w.Write(m)
		w.Flush()
		b := append([]byte(nil), buf.Bytes()...)
		out = append(out, bytes.TrimSuffix(b, []byte{0x00, 0x00, 0xff, 0xff}))
	}
	return out
}

func TestPermessageDeflateWithContextTakeover(t *testing.T) {
	m1 := bytes.Repeat([]byte("signal provisioning address "), 20)
	m2 := append([]byte("again: "), m1...) // refers back into m1's window
	var stream bytes.Buffer
	for _, c := range deflateMessages(t, m1, m2) {
		stream.Write(serverFrame(true, true, opBinary, c))
	}
	ws := newWSConn(&stream, io.Discard)
	for _, want := range [][]byte{m1, m2} {
		_, got, err := ws.readFrame()
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("got %q, %v", got, err)
		}
	}
}

func TestWriteFrameLengthEncodings(t *testing.T) {
	for _, n := range []int{0, 125, 126, 65535, 65536, 70000} {
		var out bytes.Buffer
		ws := newWSConn(bytes.NewReader(nil), &out)
		payload := bytes.Repeat([]byte{0x5a}, n)
		if err := ws.writeFrame(opBinary, payload); err != nil {
			t.Fatal(err)
		}
		b := out.Bytes()
		var got, off int
		switch l := int(b[1] & 0x7f); {
		case l < 126:
			got, off = l, 2
		case l == 126:
			got, off = int(binary.BigEndian.Uint16(b[2:4])), 4
		default:
			got, off = int(binary.BigEndian.Uint64(b[2:10])), 10
		}
		if b[0] != 0x82 || b[1]&0x80 == 0 || got != n || len(b) != off+4+n {
			t.Fatalf("n=%d: header % x decodes to length %d (frame %d bytes)", n, b[:off], got, len(b))
		}
		// round trip through the reader, which tolerates masked frames
		_, back, err := newWSConn(bytes.NewReader(b), io.Discard).readFrame()
		if err != nil || !bytes.Equal(back, payload) {
			t.Fatalf("n=%d: round trip failed: %v", n, err)
		}
	}
}
