package signalclient

import (
	"bytes"
	"io"
	"testing"
)

// Server-controlled bytes must never panic the parsers or exceed their size limits.

func FuzzReadFrame(f *testing.F) {
	f.Add(serverFrame(true, false, opBinary, []byte("hello")))
	f.Add(append(serverFrame(false, false, opBinary, []byte("ab")), serverFrame(true, false, opContinuation, []byte("cd"))...))
	f.Add(serverFrame(true, true, opBinary, []byte{0x2a, 0xce, 0xc8, 0xc9, 0xc9, 0x07, 0x00}))
	f.Add(serverFrame(true, false, opPing, bytes.Repeat([]byte{1}, 125)))
	f.Add([]byte{0x82, 127, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, stream []byte) {
		ws := newWSConn(bytes.NewReader(stream), io.Discard)
		ws.deflate = true // so compressed messages are parsed too
		for i := 0; i < 64; i++ {
			_, payload, err := ws.readFrame()
			if err != nil {
				return
			}
			if len(payload) > maxFramePayload {
				t.Fatalf("payload of %d bytes exceeds the %d limit", len(payload), maxFramePayload)
			}
			if len(ws.window) > 32768 {
				t.Fatalf("deflate window grew to %d bytes", len(ws.window))
			}
		}
	})
}

func FuzzParseMessage(f *testing.F) {
	f.Add((&wsRequest{Verb: "PUT", Path: "/v1/address", Body: []byte{0x0a, 1, 'x'}, ID: 7}).marshal())
	f.Add((&wsResponse{ID: 1, Status: 200, Message: "OK", Headers: []string{"X-Signal-Timestamp:1"}}).marshal())
	f.Add([]byte{0x08, 0x01, 0x12, 0x00})
	f.Fuzz(func(t *testing.T, b []byte) {
		msg, err := parseMessage(b)
		if err == nil && msg.Request == nil && msg.Response == nil {
			t.Fatal("nil error but neither request nor response")
		}
		if err == nil && msg.Request != nil {
			_, _ = provisioningAddress(msg.Request.Body)
		}
	})
}

func FuzzHelloSpec(f *testing.F) {
	f.Add(rawHelloLibsignal)
	f.Add(rawHelloAndroidWebSocket)
	f.Add(rawHelloDesktop)
	f.Fuzz(func(t *testing.T, raw []byte) {
		_, _ = Hello{Name: "fuzz", raw: raw}.Spec() // must not panic on any input
	})
}
