package signalclient

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// upgradeAgainst runs websocketH1 against a server that answers with head(the correct
// Sec-WebSocket-Accept value) followed by after, in one write.
func upgradeAgainst(t *testing.T, head func(accept string) string, after []byte) (*upgradeResult, error) {
	t.Helper()
	client, server := net.Pipe()
	t.Cleanup(func() { client.Close() })
	go func() {
		defer server.Close()
		req, err := http.ReadRequest(bufio.NewReader(server))
		if err != nil {
			return
		}
		server.Write(append([]byte(head(websocketAccept(req.Header.Get("Sec-WebSocket-Key")))), after...))
		time.Sleep(time.Second)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return websocketH1(ctx, client, "chat.signal.org", "/v1/websocket/provisioning/", "test")
}

// Bytes the server sends right after the 101 (often in the same TCP segment as the headers) must
// reach the WebSocket reader: they sit in the bufio.Reader that parsed the response.
func TestWebsocketH1KeepsBytesBufferedAfterThe101(t *testing.T) {
	addr := append([]byte{0x0a, 24}, []byte("abcdefghijklmnopqrstuvwx")...)
	frame := serverFrame(true, false, opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/address", Body: addr, ID: 7}).marshal())
	up, err := upgradeAgainst(t, func(accept string) string {
		return "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\nX-Signal-Timestamp: 1\r\n\r\n"
	}, frame)
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

// RFC 6455 section 1.3's example.
func TestWebsocketAccept(t *testing.T) {
	if got := websocketAccept("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatal(got)
	}
}

// A 101 counts as an upgrade only with the headers RFC 6455 section 4.1 requires. Header names and
// the Upgrade and Connection tokens are case-insensitive, and Connection may list other tokens.
func TestWebsocketH1ChecksThe101(t *testing.T) {
	for name, tc := range map[string]struct {
		head string
		ok   bool
	}{
		"valid":              {"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %ACCEPT%\r\n", true},
		"other case":         {"upgrade: WebSocket\r\nconnection: keep-alive, upgrade\r\nsec-websocket-accept: %ACCEPT%\r\n", true},
		"no accept":          {"Upgrade: websocket\r\nConnection: Upgrade\r\n", false},
		"wrong accept":       {"Upgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n", false},
		"no upgrade":         {"Connection: Upgrade\r\nSec-WebSocket-Accept: %ACCEPT%\r\n", false},
		"no connection":      {"Upgrade: websocket\r\nSec-WebSocket-Accept: %ACCEPT%\r\n", false},
		"upgrade to another": {"Upgrade: h2c\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %ACCEPT%\r\n", false},
	} {
		up, err := upgradeAgainst(t, func(accept string) string {
			return "HTTP/1.1 101 Switching Protocols\r\n" + strings.Replace(tc.head, "%ACCEPT%", accept, 1) + "\r\n"
		}, nil)
		switch {
		case tc.ok && err != nil:
			t.Errorf("%s: %v", name, err)
		case !tc.ok && !errors.Is(err, errUpgradeInvalid):
			t.Errorf("%s: expected errUpgradeInvalid, got %v", name, err)
		case !tc.ok && (up == nil || up.status != 101):
			t.Errorf("%s: the 101 status was not recorded: %+v", name, up)
		}
	}
}
