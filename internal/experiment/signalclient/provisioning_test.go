package signalclient

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// fakeServer plays Signal's side of a provisioning socket on conn.
type fakeServer struct {
	sendAddress     bool
	closeAtStart    bool
	noTimestamp     bool
	badPong         bool
	pingAfterAck    bool         // send a PING right after the client's ack (during idle)
	gotPong         *atomic.Bool // set when the client answers that PING
	timestampHeader string       // replaces the X-Signal-Timestamp header line when set

	spamPongs          bool   // send unsolicited pongs every 50 ms instead of the address
	closeOnPing        bool   // answer the client's PING with a close frame (TCP stays open)
	junkPongFirst      bool   // answer the client's PING with an unrelated pong, then the echo
	keepaliveWrongID   bool   // answer keepalives with a different id
	keepaliveStatus    uint32 // answer keepalives with this status (default 200)
	compress           bool   // send every message with permessage-deflate (RSV1, shared context)
	corruptCompressed  bool   // send the address as an RSV1 frame that does not inflate
	addressWithoutAddr bool   // send a ProvisioningAddress with no address field
	oversizedFrame     bool   // send a frame header announcing more than maxFramePayload
	reservedOpcode     bool   // send a frame with reserved opcode 0x3 instead of the address
	stopAfterAddress   bool   // return right after sending the address (the caller then kills the stream)
	extraAfterAddress  bool   // send another binary message right after the address
	ignoreKeepalives   bool   // stay connected but never answer keepalives
}

func (f fakeServer) run(t *testing.T, conn net.Conn) {
	defer conn.Close()
	f.serve(t, conn, conn)
}

// serve runs the server side of the provisioning flow over any byte stream (an HTTP/1.1
// connection after the upgrade, or an HTTP/2 stream), until the client closes.
func (f fakeServer) serve(t *testing.T, r io.Reader, w io.Writer) {
	if f.closeAtStart {
		return
	}
	ws := newWSConn(r, w) // reads the client's masked frames
	var deflated bytes.Buffer
	deflater, _ := flate.NewWriter(&deflated, flate.BestSpeed)
	send := func(op byte, p []byte) {
		if f.compress && op == opBinary {
			deflated.Reset()
			deflater.Write(p)
			deflater.Flush()
			w.Write(serverFrame(true, true, op, bytes.TrimSuffix(deflated.Bytes(), []byte{0, 0, 0xff, 0xff})))
			return
		}
		w.Write(serverFrame(true, false, op, p))
	}
	switch {
	case f.spamPongs:
		for i := 0; i < 40; i++ {
			send(opPong, []byte("unsolicited"))
			time.Sleep(50 * time.Millisecond)
		}
		return
	case f.oversizedFrame:
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], maxFramePayload+1)
		w.Write(append([]byte{0x82, 127}, l[:]...))
		return
	case f.reservedOpcode:
		w.Write(serverFrame(true, false, 0x3, []byte("x")))
		return
	case f.corruptCompressed:
		w.Write(serverFrame(true, true, opBinary, []byte{0xff, 0xff, 0xff, 0xff, 0x00}))
		return
	case f.addressWithoutAddr:
		send(opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/address", Body: []byte{0x10, 0x01}, ID: 7}).marshal())
	case f.sendAddress:
		addr := append([]byte{0x0a, 24}, []byte("abcdefghijklmnopqrstuvwx")...)
		send(opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/address", Body: addr, ID: 7}).marshal())
		if f.extraAfterAddress {
			send(opBinary, (&wsRequest{Verb: "PUT", Path: "/v1/message", ID: 8}).marshal())
		}
		if f.stopAfterAddress {
			return
		}
	}
	for {
		op, p, err := ws.readFrame()
		if err != nil {
			return
		}
		switch op {
		case opPing:
			switch {
			case f.closeOnPing:
				send(opClose, []byte{0x03, 0xe8})
				continue
			case f.junkPongFirst:
				send(opPong, []byte("not the payload"))
			case f.badPong:
				p = []byte("not the payload")
			}
			send(opPong, p)
		case opPong:
			if f.gotPong != nil && string(p) == "server-ping" {
				f.gotPong.Store(true)
			}
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
			if msg.Response != nil && f.pingAfterAck {
				send(opPing, []byte("server-ping"))
			}
			if msg.Request != nil && msg.Request.Path == "/v1/keepalive" && !f.ignoreKeepalives {
				h := []string{"X-Signal-Timestamp:1791563494213", "Content-Length:0"}
				if f.timestampHeader != "" {
					h[0] = f.timestampHeader
				}
				if f.noTimestamp {
					h = h[1:]
				}
				id, status := msg.Request.ID, f.keepaliveStatus
				if f.keepaliveWrongID {
					id++
				}
				if status == 0 {
					status = 200
				}
				send(opBinary, (&wsResponse{ID: id, Status: status, Message: "OK", Headers: h}).marshal())
			}
		}
	}
}

func runAgainst(t *testing.T, f fakeServer) *ProvisioningResult {
	t.Helper()
	return runAgainstIdle(t, f, 0)
}

func runAgainstIdle(t *testing.T, f fakeServer, idle time.Duration) *ProvisioningResult {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	go f.run(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws := newWSConn(client, client)
	ws.deflate = f.compress || f.corruptCompressed // as if negotiated
	return runProvisioning(ctx, ws, time.Now(), []int{0, 1024, 16384}, idle, nil)
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
	defer func(d time.Duration) { keepaliveTimeout = d }(keepaliveTimeout)
	keepaliveTimeout = 300 * time.Millisecond
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

// A PING during idle must be answered and must not cut the idle short.
func TestProvisioningIdleSurvivesServerPing(t *testing.T) {
	var gotPong atomic.Bool
	idle := 300 * time.Millisecond
	r := runAgainstIdle(t, fakeServer{sendAddress: true, pingAfterAck: true, gotPong: &gotPong}, idle)
	if r.Failure != nil {
		t.Fatal(deref(r.Failure))
	}
	if !gotPong.Load() {
		t.Error("the PING sent during idle was not answered")
	}
	if waited := r.Keepalives[0].T0 - r.AddressT; waited < idle.Seconds() {
		t.Errorf("keepalives started %.3f s after the address, before the %v idle ended", waited, idle)
	}
}

// Only a header named exactly X-Signal-Timestamp counts.
func TestProvisioningTimestampHeaderNameIsExact(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, timestampHeader: "X-Signal-Timestamp-Foo:1"})
	if r.Failure != nil {
		t.Fatal(deref(r.Failure))
	}
	for _, k := range r.Keepalives {
		if k.SignalTimestamp {
			t.Fatalf("X-Signal-Timestamp-Foo counted as the timestamp: %+v", k)
		}
	}
}

func TestTargetTimeoutCoversEveryStep(t *testing.T) {
	steps := addressTimeout + time.Duration(len(keepalivePaddings)+1)*keepaliveTimeout
	for _, m := range []Method{MethodProvisioningH1, MethodProvisioningH2} {
		if got := targetTimeout(m, 7*time.Second); got < connectBudget+steps+7*time.Second {
			t.Errorf("%s: budget %v does not cover connect (%v) + steps (%v) + idle", m, got, connectBudget, steps)
		}
	}
	if got := targetTimeout(MethodHTTPSGet, time.Hour); got != connectBudget {
		t.Errorf("https_get budget %v should not include idle", got)
	}
}

// Ignored frames must not extend a step: the address step fails at its deadline even though
// the server keeps sending unsolicited pongs.
func TestProvisioningStepDeadlineIsNotExtendedByIgnoredFrames(t *testing.T) {
	defer func(d time.Duration) { addressTimeout = d }(addressTimeout)
	addressTimeout = 300 * time.Millisecond
	start := time.Now()
	r := runAgainst(t, fakeServer{spamPongs: true})
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "generic_timeout_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("the step took %v despite its %v deadline", elapsed, addressTimeout)
	}
}

func TestProvisioningCloseDuringPing(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, closeOnPing: true})
	if r.FailedStep != "ping" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningUnsolicitedPongBeforeEcho(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, junkPongFirst: true})
	if r.Failure != nil || r.PingEchoed == nil || !*r.PingEchoed {
		t.Fatalf("expected the echo to be found after a junk pong: echoed=%v failure=%q", r.PingEchoed, deref(r.Failure))
	}
}

func TestProvisioningKeepaliveWrongIDAndStatus(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, keepaliveWrongID: true, keepaliveStatus: 500})
	if r.Failure != nil {
		t.Fatal(deref(r.Failure))
	}
	for _, k := range r.Keepalives {
		if k.IDMatch || k.Status != 500 {
			t.Fatalf("expected id mismatch and status 500, got %+v", k)
		}
	}
}

func TestProvisioningCompressedMessages(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, compress: true})
	if r.Failure != nil || !r.AddressReceived || len(r.Keepalives) != 3 || !r.Keepalives[2].SignalTimestamp {
		t.Fatalf("compressed flow failed: %+v failure=%q", r, deref(r.Failure))
	}
}

func TestProvisioningMalformedInputs(t *testing.T) {
	for name, f := range map[string]fakeServer{
		"corrupt deflate":            {corruptCompressed: true},
		"address without address":    {addressWithoutAddr: true},
		"frame above the size limit": {oversizedFrame: true},
		"reserved opcode":            {reservedOpcode: true},
	} {
		r := runAgainst(t, f)
		if r.FailedStep != "provisioning_address" || deref(r.Failure) != FailureMalformedMessage {
			t.Errorf("%s: got %q/%q", name, r.FailedStep, deref(r.Failure))
		}
	}
}

func TestProvisioningKeepaliveFailureSetsT(t *testing.T) {
	defer func(d time.Duration) { keepaliveTimeout = d }(keepaliveTimeout)
	keepaliveTimeout = 200 * time.Millisecond
	r := runAgainst(t, fakeServer{sendAddress: true, ignoreKeepalives: true})
	if r.FailedStep != "keepalive" || len(r.Keepalives) != 1 {
		t.Fatalf("expected the first keepalive to fail, got step %q with %d keepalives", r.FailedStep, len(r.Keepalives))
	}
	if k := r.Keepalives[0]; k.Failure == nil || k.T < k.T0+0.15 {
		t.Fatalf("failed keepalive: failure=%q t=%v t0=%v", deref(k.Failure), k.T, k.T0)
	}
}

// A failed write is explained by the read error behind it even when other messages arrived first:
// the server sends a message after the address and closes, so the ack's write fails, and the EOF
// queued behind that message is the reason.
func TestProvisioningWriteFailureSkipsQueuedMessages(t *testing.T) {
	r := runAgainst(t, fakeServer{sendAddress: true, extraAfterAddress: true, stopAfterAddress: true})
	if r.FailedStep != "ack" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
}
