package signalclient

import (
	"context"
	"net"
	"testing"
	"time"
)

// fakeServer plays Signal's side of a provisioning socket on conn.
type fakeServer struct {
	sendAddress  bool
	closeAtStart bool
	noTimestamp  bool
	badPong      bool
}

func (f fakeServer) run(t *testing.T, conn net.Conn) {
	defer conn.Close()
	if f.closeAtStart {
		return
	}
	ws := newWSConn(conn, conn) // reads the client's masked frames
	send := func(op byte, p []byte) { conn.Write(serverFrame(true, false, op, p)) }
	if f.sendAddress {
		addr := append([]byte{0x0a, 24}, []byte("abcdefghijklmnopqrstuvwx")...)
		send(opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/address", Body: addr, ID: 7}).marshal())
	}
	for {
		op, p, err := ws.readFrame()
		if err != nil {
			return
		}
		switch op {
		case opPing:
			if f.badPong {
				p = []byte("not the payload")
			}
			send(opPong, p)
		case opClose:
			return
		case opBinary:
			msg, err := parseMessage(p)
			if err != nil {
				t.Errorf("server: %v", err)
				return
			}
			if msg.Response != nil && (msg.Response.ID != 7 || msg.Response.Status != 200) {
				t.Errorf("bad ack: %+v", msg.Response)
			}
			if msg.Request != nil && msg.Request.Path == "/v1/keepalive" {
				h := []string{"X-Signal-Timestamp:1791563494213", "Content-Length:0"}
				if f.noTimestamp {
					h = h[1:]
				}
				send(opBinary, (&wsResponse{ID: msg.Request.ID, Status: 200, Message: "OK", Headers: h}).marshal())
			}
		}
	}
}

func runAgainst(t *testing.T, f fakeServer) *ProvisioningResult {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	go f.run(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runProvisioning(ctx, newWSConn(client, client), time.Now(), []int{0, 1024, 16384}, 0)
}

func TestProvisioningHappyPath(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true})
	if r.Failure != nil || !r.AddressReceived || r.AddressLength != 24 || !r.Acked {
		t.Fatalf("unexpected: %+v failure=%v", r, deref(r.Failure))
	}
	if len(r.Keepalives) != 3 {
		t.Fatalf("keepalives: %d", len(r.Keepalives))
	}
	for _, k := range r.Keepalives {
		if k.Status != 200 || !k.IDMatch || !k.SignalTimestamp || k.Failure != nil {
			t.Fatalf("keepalive %+v", k)
		}
	}
	if r.PingEchoed == nil || !*r.PingEchoed {
		t.Fatal("ping not echoed")
	}
}

func TestProvisioningNoAddress(t *testing.T) {
	defer func(d time.Duration) { addressTimeout = d }(addressTimeout)
	addressTimeout = 200 * time.Millisecond
	r := runAgainst(t, fakeServer{})
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "generic_timeout_error" {
		t.Fatalf("got step %q failure %q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningClosedAfterUpgrade(t *testing.T) {
	r := runAgainst(t, fakeServer{closeAtStart: true})
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got step %q failure %q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningReplyWithoutTimestamp(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, noTimestamp: true})
	if r.Failure != nil {
		t.Fatal(deref(r.Failure))
	}
	for _, k := range r.Keepalives {
		if k.SignalTimestamp {
			t.Fatal("timestamp reported although the reply had none")
		}
	}
}

func TestProvisioningPongMismatch(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, badPong: true})
	if r.PingEchoed == nil || *r.PingEchoed {
		t.Fatalf("expected ping_echoed=false, got %v", r.PingEchoed)
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
