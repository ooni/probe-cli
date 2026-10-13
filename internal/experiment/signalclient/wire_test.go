package signalclient

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/signalclient/signalpb"
	"google.golang.org/protobuf/proto"
)

// The messages the client sends, byte for byte: proto2 sends id even when it is 0, and fields go
// in field-number order.
func TestMessagesWeSend(t *testing.T) {
	keepalive := requestMessage(&signalpb.WebSocketRequestMessage{
		Verb: proto.String("GET"), Path: proto.String("/v1/keepalive"), Id: proto.Uint64(0)})
	want := "0801" + "1216" + "0a03" + hex.EncodeToString([]byte("GET")) + "120d" +
		hex.EncodeToString([]byte("/v1/keepalive")) + "2000"
	if got := hex.EncodeToString(keepalive); got != want {
		t.Errorf("keepalive:\n got %s\nwant %s", got, want)
	}
	ack := responseMessage(&signalpb.WebSocketResponseMessage{
		Id: proto.Uint64(7), Status: proto.Uint32(200), Message: proto.String("OK")})
	want = "0802" + "1a09" + "0807" + "10c801" + "1a02" + hex.EncodeToString([]byte("OK"))
	if got := hex.EncodeToString(ack); got != want {
		t.Errorf("ack:\n got %s\nwant %s", got, want)
	}
}

func TestParseMessage(t *testing.T) {
	msg, err := parseMessage(serverRequest("PUT", "/v1/address", 7, addressBody()))
	if err != nil || msg.GetType() != signalpb.WebSocketMessage_REQUEST || msg.GetRequest().GetPath() != "/v1/address" ||
		msg.GetRequest().GetId() != 7 {
		t.Fatalf("request: %v %v", err, msg)
	}
	headers := []string{"X-Signal-Timestamp:1"}
	msg, err = parseMessage(responseMessage(&signalpb.WebSocketResponseMessage{
		Id: proto.Uint64(42), Status: proto.Uint32(200), Message: proto.String("OK"), Headers: headers}))
	if err != nil || msg.GetType() != signalpb.WebSocketMessage_RESPONSE || msg.GetResponse().GetStatus() != 200 ||
		msg.GetResponse().GetHeaders()[0] != headers[0] {
		t.Fatalf("response: %v %v", err, msg)
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
	addr, err := provisioningAddress(addressBody())
	if err != nil || addr != testAddress {
		t.Fatalf("got %q %v", addr, err)
	}
	if _, err := provisioningAddress(addressWithoutAddress); !errors.Is(err, errNoProvisioningAddress) {
		t.Fatalf("without an address: %v", err)
	}
}

// testAddress is the provisioning address the tests' servers send.
const testAddress = "abcdefghijklmnopqrstuvwx"

// addressWithoutAddress is a ProvisioningAddress body with only an unknown field (2 = 1).
var addressWithoutAddress = []byte{0x10, 0x01}

// addressBody is a ProvisioningAddress carrying testAddress.
func addressBody() []byte {
	return marshal(&signalpb.ProvisioningAddress{Address: proto.String(testAddress)})
}

// serverRequest encodes a request as the server sends it.
func serverRequest(verb, path string, id uint64, body []byte) []byte {
	return requestMessage(&signalpb.WebSocketRequestMessage{
		Verb: proto.String(verb), Path: proto.String(path), Id: proto.Uint64(id), Body: body})
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

// Frames RFC 6455 and RFC 7692 forbid are errors, as in the apps' WebSocket clients.
func TestReadFrameRejectsProtocolViolations(t *testing.T) {
	ping126 := serverFrame(true, false, opPing, bytes.Repeat([]byte{1}, 126))
	rsv2 := serverFrame(true, false, opBinary, []byte("x"))
	rsv2[0] |= 0x20
	rsv3 := serverFrame(true, false, opBinary, []byte("x"))
	rsv3[0] |= 0x10
	for name, tc := range map[string]struct {
		stream  []byte
		deflate bool
	}{
		"control frame over 125 bytes": {ping126, false},
		"fragmented control frame":     {serverFrame(false, false, opPing, []byte("p")), false},
		"RSV2":                         {rsv2, true},
		"RSV3":                         {rsv3, true},
		"RSV1 without deflate":         {serverFrame(true, true, opBinary, []byte{0x02, 0x00}), false},
		"RSV1 on a control frame":      {serverFrame(true, true, opPing, []byte("p")), true},
		"RSV1 on a continuation": {append(serverFrame(false, true, opBinary, []byte{0x02}),
			serverFrame(true, true, opContinuation, []byte{0x00})...), true},
		"reserved data opcode":    {serverFrame(true, false, 0x3, []byte("x")), false},
		"reserved control opcode": {serverFrame(true, false, 0xB, []byte("x")), false},
		"continuation first":      {serverFrame(true, false, opContinuation, []byte("x")), false},
		"new message mid-message": {append(serverFrame(false, false, opBinary, []byte("ab")),
			serverFrame(true, false, opBinary, []byte("cd"))...), false},
	} {
		ws := newWSConn(bytes.NewReader(tc.stream), io.Discard)
		ws.deflate = tc.deflate
		var err error
		for err == nil {
			_, _, err = ws.readFrame()
		}
		if !errors.Is(err, errProtocolViolation) {
			t.Errorf("%s: got %v", name, err)
		}
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
	ws.deflate = true
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
