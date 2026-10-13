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

	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/experiment/signalclient/signalpb"
	"google.golang.org/protobuf/proto"
)

// fault is how a fake Signal server misbehaves; the zero value behaves like Signal. The provisioning
// socket's faults are fakeServer's; the front ends' (qa_servers_test.go) also pass them through.
type fault int

const (
	_ fault = iota // behaves like Signal

	// the provisioning socket
	faultNoAddress             // the provisioning address never arrives
	faultCloseAtStart          // the stream ends right after the upgrade
	faultNoTimestamp           // replies lack X-Signal-Timestamp, as injected replies would
	faultTimestampLookalike    // replies carry X-Signal-Timestamp-Foo instead of X-Signal-Timestamp
	faultSpamPongs             // unsolicited pongs every 50 ms instead of the address
	faultAddressWithoutAddress // a ProvisioningAddress with no address field
	faultCorruptCompressed     // the address as a compressed (RSV1) frame that does not inflate
	faultOversizedFrame        // a frame header announcing more than maxFramePayload
	faultReservedOpcode        // a frame with reserved opcode 0x3 instead of the address
	faultMessageThenClose      // another message right after the address, then the stream ends
	faultIgnoreKeepalives      // never answer keepalives
	faultKillLargeMessages     // end the stream on a message above 8 KiB, as a size-triggered middlebox would
	faultKeepaliveWrongAnswer  // answer keepalives with another id and status 500
	faultBadPong               // answer the client's PING with another payload
	faultJunkPongFirst         // answer the client's PING with an unrelated pong, then the echo
	faultCloseOnPing           // answer the client's PING with a close frame (TCP stays open)

	// the front ends
	faultResetAfterAddress    // HTTP/2: RST_STREAM right after the address
	faultKillAfterAddress     // close the TCP connection right after the address
	faultNoConnectProtocol    // HTTP/2 SETTINGS without ENABLE_CONNECT_PROTOCOL
	faultRefuseUpgrade        // 403 to the provisioning upgrade, as a CDN's geoblock would
	faultBadAccept            // a 101 with the wrong Sec-WebSocket-Accept (HTTP/1.1)
	faultEnclaveAccepts       // accept enclave upgrades without credentials (101)
	faultEnclave401           // 401 to enclave upgrades, as for a real enclave id without credentials
	faultTruncateBody         // GET responses: promise 1000 bytes, send 5, close the connection
	faultResetRequest         // close the connection when a GET arrives, before any response
	faultImpostorCertificate  // a self-signed certificate for the requested name, as an intercepting middlebox's
	faultBlockpageCertificate // a self-signed certificate for a blockpage domain, as many middleboxes present
	deflate                   // not a fault: accept permessage-deflate (HTTP/1.1), compress every message
)

// fakeServer plays Signal's side of a provisioning socket.
type fakeServer struct {
	fault    fault
	compress bool // permessage-deflate was negotiated: send every message compressed

	// pingAfterAck, when set, makes the server PING right after the client's ack (during the
	// idle time), and records whether the client answered.
	pingAfterAck *atomic.Bool
}

func (f fakeServer) run(t *testing.T, conn net.Conn) {
	defer conn.Close()
	f.serve(t, conn, conn)
}

// serve runs the server side of the provisioning flow over any byte stream (an HTTP/1.1
// connection after the upgrade, or an HTTP/2 stream), until the client closes.
func (f fakeServer) serve(t *testing.T, r io.Reader, w io.Writer) {
	if f.fault == faultCloseAtStart {
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
	switch f.fault {
	case faultNoAddress:
	case faultSpamPongs:
		for i := 0; i < 40; i++ {
			send(opPong, []byte("unsolicited"))
			time.Sleep(50 * time.Millisecond)
		}
		return
	case faultOversizedFrame:
		var l [8]byte
		binary.BigEndian.PutUint64(l[:], maxFramePayload+1)
		w.Write(append([]byte{0x82, 127}, l[:]...))
		return
	case faultReservedOpcode:
		w.Write(serverFrame(true, false, 0x3, []byte("x")))
		return
	case faultCorruptCompressed:
		w.Write(serverFrame(true, true, opBinary, []byte{0xff, 0xff, 0xff, 0xff, 0x00}))
		return
	case faultAddressWithoutAddress:
		send(opBinary, serverRequest("PUT", "/v1/address", 7, addressWithoutAddress))
	default:
		send(opBinary, serverRequest("PUT", "/v1/address", 7, addressBody()))
		switch f.fault {
		case faultMessageThenClose:
			send(opBinary, serverRequest("PUT", "/v1/message", 8, nil))
			return
		case faultResetAfterAddress, faultKillAfterAddress: // the front end resets or kills
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
			switch f.fault {
			case faultCloseOnPing:
				send(opClose, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				continue
			case faultJunkPongFirst:
				send(opPong, []byte("not the payload"))
			case faultBadPong:
				p = []byte("not the payload")
			}
			send(opPong, p)
		case opPong:
			if f.pingAfterAck != nil && string(p) == "server-ping" {
				f.pingAfterAck.Store(true)
			}
		case opClose:
			return
		case opBinary:
			if f.fault == faultKillLargeMessages && len(p) > 8192 {
				return
			}
			msg, err := parseMessage(p)
			if err != nil {
				t.Errorf("server: %v", err)
				return
			}
			if msg.Response != nil && (msg.Response.GetId() != 7 || msg.Response.GetStatus() != 200) {
				t.Errorf("bad ack: %+v", msg.Response)
			}
			if msg.Response != nil && f.pingAfterAck != nil {
				send(opPing, []byte("server-ping"))
			}
			if msg.Request.GetPath() == "/v1/keepalive" && f.fault != faultIgnoreKeepalives {
				send(opBinary, f.keepaliveAnswer(msg.Request.GetId()))
			}
		}
	}
}

// keepaliveAnswer is the server's response to the keepalive with id.
func (f fakeServer) keepaliveAnswer(id uint64) []byte {
	resp := &signalpb.WebSocketResponseMessage{Id: proto.Uint64(id), Status: proto.Uint32(200), Message: proto.String("OK"),
		Headers: []string{"X-Signal-Timestamp:1791563494213", "Content-Length:0"}}
	switch f.fault {
	case faultNoTimestamp:
		resp.Headers = resp.Headers[1:]
	case faultTimestampLookalike:
		resp.Headers[0] = "X-Signal-Timestamp-Foo:1"
	case faultKeepaliveWrongAnswer:
		resp.Id, resp.Status = proto.Uint64(id+1), proto.Uint32(500)
	}
	return responseMessage(resp)
}

// runAgainst runs the flow against f, with the measurement's flow changed by adjust.
func runAgainst(t *testing.T, f fakeServer, adjust ...func(*provisioningFlow)) *ProvisioningResult {
	t.Helper()
	flow := newMeasurer(Config{}).provisioningFlow()
	for _, fn := range adjust {
		fn(&flow)
	}
	client, server := net.Pipe()
	defer client.Close()
	go f.run(t, server)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws := newWSConn(client, client)
	ws.deflate = f.compress || f.fault == faultCorruptCompressed // as if negotiated
	return runProvisioning(ctx, ws, time.Now(), flow, nil)
}

func TestProvisioningHappyPath(t *testing.T) {
	r := runAgainst(t, fakeServer{})
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
	r := runAgainst(t, fakeServer{fault: faultNoAddress}, func(flow *provisioningFlow) { flow.addressTimeout = 200 * time.Millisecond })
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "generic_timeout_error" {
		t.Fatalf("got step %q failure %q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningClosedAfterUpgrade(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultCloseAtStart})
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got step %q failure %q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningReplyWithoutTimestamp(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultNoTimestamp})
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
	r := runAgainst(t, fakeServer{fault: faultBadPong},
		func(flow *provisioningFlow) { flow.keepaliveTimeout = 300 * time.Millisecond })
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
	r := runAgainst(t, fakeServer{pingAfterAck: &gotPong},
		func(flow *provisioningFlow) { flow.idle = idle })
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
	r := runAgainst(t, fakeServer{fault: faultTimestampLookalike})
	if r.Failure != nil {
		t.Fatal(deref(r.Failure))
	}
	for _, k := range r.Keepalives {
		if k.SignalTimestamp {
			t.Fatalf("X-Signal-Timestamp-Foo counted as the timestamp: %+v", k)
		}
	}
}

// Ignored frames must not extend a step: the address step fails at its deadline even though
// the server keeps sending unsolicited pongs.
func TestProvisioningStepDeadlineIsNotExtendedByIgnoredFrames(t *testing.T) {
	const deadline = 300 * time.Millisecond
	start := time.Now()
	r := runAgainst(t, fakeServer{fault: faultSpamPongs}, func(flow *provisioningFlow) { flow.addressTimeout = deadline })
	if r.FailedStep != "provisioning_address" || deref(r.Failure) != "generic_timeout_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("the step took %v despite its %v deadline", elapsed, deadline)
	}
}

func TestProvisioningCloseDuringPing(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultCloseOnPing})
	if r.FailedStep != "ping" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
}

func TestProvisioningUnsolicitedPongBeforeEcho(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultJunkPongFirst})
	if r.Failure != nil || r.PingEchoed == nil || !*r.PingEchoed {
		t.Fatalf("expected the echo to be found after a junk pong: echoed=%v failure=%q", r.PingEchoed, deref(r.Failure))
	}
}

func TestProvisioningKeepaliveWrongIDAndStatus(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultKeepaliveWrongAnswer})
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
	r := runAgainst(t, fakeServer{compress: true})
	if r.Failure != nil || !r.AddressReceived || len(r.Keepalives) != 3 || !r.Keepalives[2].SignalTimestamp {
		t.Fatalf("compressed flow failed: %+v failure=%q", r, deref(r.Failure))
	}
}

func TestProvisioningMalformedInputs(t *testing.T) {
	for name, fault := range map[string]fault{
		"corrupt deflate":            faultCorruptCompressed,
		"address without address":    faultAddressWithoutAddress,
		"frame above the size limit": faultOversizedFrame,
		"reserved opcode":            faultReservedOpcode,
	} {
		r := runAgainst(t, fakeServer{fault: fault})
		if r.FailedStep != "provisioning_address" || deref(r.Failure) != FailureMalformedMessage {
			t.Errorf("%s: got %q/%q", name, r.FailedStep, deref(r.Failure))
		}
	}
}

func TestProvisioningKeepaliveFailureSetsT(t *testing.T) {
	r := runAgainst(t, fakeServer{fault: faultIgnoreKeepalives},
		func(flow *provisioningFlow) { flow.keepaliveTimeout = 200 * time.Millisecond })
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
	r := runAgainst(t, fakeServer{fault: faultMessageThenClose})
	if r.FailedStep != "ack" || deref(r.Failure) != "eof_error" {
		t.Fatalf("got %q/%q", r.FailedStep, deref(r.Failure))
	}
}
