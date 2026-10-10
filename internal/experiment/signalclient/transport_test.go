package signalclient

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// Bytes the server sends right after the 101 (often in the same TCP segment as the headers) must
// reach the WebSocket reader: they sit in the bufio.Reader that parsed the response.
func TestWebsocketH1KeepsBytesBufferedAfterThe101(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		if _, err := http.ReadRequest(bufio.NewReader(server)); err != nil {
			return
		}
		addr := append([]byte{0x0a, 24}, []byte("abcdefghijklmnopqrstuvwx")...)
		frame := serverFrame(true, false, opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/address", Body: addr, ID: 7}).marshal())
		head := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nX-Signal-Timestamp: 1\r\n\r\n"
		server.Write(append([]byte(head), frame...)) // one write: headers and the first frame together
		time.Sleep(time.Second)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	up, err := websocketH1(ctx, client, "chat.signal.org", "/v1/websocket/provisioning/", "test")
	if err != nil {
		t.Fatal(err)
	}
	op, payload, err := up.ws.readFrame()
	if err != nil || op != opBinary {
		t.Fatalf("op %d err %v", op, err)
	}
	msg, err := parseMessage(payload)
	if err != nil || msg.Request == nil || msg.Request.Path != "/v1/address" {
		t.Fatalf("expected the buffered address frame, got %+v %v", msg, err)
	}
}
