package signalclient

//
// The pre-account provisioning flow: what a freshly installed Signal client does when it shows
// the "link this device" (Desktop, iOS) or "transfer from old device" (Android) QR code.
//
//  1. the server pushes REQUEST PUT /v1/address (a ProvisioningAddress) right after the upgrade;
//     Android gives up if it does not arrive within 10 s (ProvisioningSocket.kt)
//  2. the client acks it with RESPONSE 200 "OK"
//  3. the client sends GET /v1/keepalive requests (Android: every 30 s); we pad some of them so
//     message-sized traffic crosses the session, and match each response by id
//  4. a WebSocket PING with a random payload, which the PONG must echo
//
// No account or phone number is involved and nothing is created on Signal's side: the
// provisioning address expires unused. The address is a short-lived capability, so it is never
// recorded, only its length.
//

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/experiment/signalclient/signalpb"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"google.golang.org/protobuf/proto"
)

// ProvisioningResult is what happened on one provisioning socket.
type ProvisioningResult struct {
	AddressReceived bool               `json:"address_received"`
	AddressLength   int                `json:"address_length"`
	AddressT        float64            `json:"address_t"`
	Acked           bool               `json:"acked"`
	Keepalives      []*KeepaliveResult `json:"keepalives"`
	PingEchoed      *bool              `json:"ping_echoed"`
	IdleSeconds     int64              `json:"idle_seconds"`
	Failure         *string            `json:"failure"`
	FailedStep      string             `json:"failed_step,omitempty"`
}

// KeepaliveResult is one GET /v1/keepalive round trip.
type KeepaliveResult struct {
	Padding         int     `json:"padding"`
	T0              float64 `json:"t0"`
	T               float64 `json:"t"`
	Status          int64   `json:"status"`
	IDMatch         bool    `json:"id_match"`
	SignalTimestamp bool    `json:"x_signal_timestamp"`
	Failure         *string `json:"failure"`
}

type frame struct {
	op      byte
	payload []byte
	err     error
}

// pendingReadErrorWait is how long a failed write waits for the read error that explains it.
const pendingReadErrorWait = 200 * time.Millisecond

// errStepTimeout is a step that did not complete in time (classified as generic_timeout_error).
var errStepTimeout = fmt.Errorf("signal_client: provisioning step: %w", context.DeadlineExceeded)

// provisioningFlow is how runProvisioning runs the flow.
type provisioningFlow struct {
	paddings         []int         // the keepalives' body sizes
	idle             time.Duration // how long to stay quiet after the ack (0: not at all)
	addressTimeout   time.Duration // waiting for the server's address
	keepaliveTimeout time.Duration // waiting for each keepalive's response, and for the PONG
}

// runProvisioning drives the flow on an upgraded WebSocket. checkpoint, when not nil, is called
// after each step (the caller drains its trace's bounded event buffers there).
func runProvisioning(ctx context.Context, ws *wsConn, zero time.Time, flow provisioningFlow,
	checkpoint func()) *ProvisioningResult {
	res := &ProvisioningResult{IdleSeconds: int64(flow.idle / time.Second)}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if checkpoint == nil {
		checkpoint = func() {}
	}
	s := newSession(ctx, ws, flow)
	fail := func(step string, err error) *ProvisioningResult {
		res.FailedStep, res.Failure = step, measurexlite.NewFailure(err)
		return res
	}

	// 1. the server's provisioning address
	req, addr, err := s.awaitAddress()
	if err != nil {
		return fail("provisioning_address", err)
	}
	res.AddressReceived, res.AddressLength, res.AddressT = true, len(addr), time.Since(zero).Seconds()

	// 2. ack, as the apps do
	if err := s.send(opBinary, responseMessage(&signalpb.WebSocketResponseMessage{
		Id: proto.Uint64(req.GetId()), Status: proto.Uint32(200), Message: proto.String("OK")})); err != nil {
		return fail("ack", err)
	}
	res.Acked = true
	checkpoint()

	if flow.idle > 0 {
		if err := s.idle(flow.idle); err != nil {
			return fail("idle", err)
		}
		checkpoint()
	}

	// 3. keepalives, some padded
	for i, pad := range flow.paddings {
		kr, err := s.keepalive(uint64(i), pad, zero) // #nosec G115 -- a few keepalives
		res.Keepalives = append(res.Keepalives, kr)
		if err != nil {
			return fail("keepalive", err)
		}
		checkpoint()
	}

	// 4. a PING whose payload a PONG must echo
	echoed, err := s.ping()
	if err != nil {
		return fail("ping", err)
	}
	res.PingEchoed = &echoed
	checkpoint()

	_ = ws.writeFrame(opClose, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")) // the flow already succeeded
	return res
}

// session is the client side of one provisioning socket: the frames read in the background,
// and the steps of the flow. Each step has one deadline, so frames it ignores cannot extend it.
type session struct {
	ctx    context.Context
	ws     *wsConn
	flow   provisioningFlow
	frames chan frame
}

// newSession starts reading ws's frames until a read error or ctx's end.
func newSession(ctx context.Context, ws *wsConn, flow provisioningFlow) *session {
	s := &session{ctx: ctx, ws: ws, flow: flow, frames: make(chan frame, 16)}
	go func() {
		for {
			op, p, err := ws.readFrame()
			if err != nil {
				err = readFailure(withCause(ws, err))
			}
			select {
			case s.frames <- frame{op, p, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return s
}

// recv returns the next frame that want accepts, answering pings on the way. A read error, a
// close from the server, the deadline (errStepTimeout) or the end of the session ends it.
func (s *session) recv(deadline time.Time, want func(frame) bool) (frame, error) {
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for {
		select {
		case f := <-s.frames:
			switch {
			case f.err != nil:
				return f, f.err
			case f.op == opClose:
				return f, eofFailure()
			case f.op == opPing:
				_ = s.ws.writeFrame(opPong, f.payload)
			case want(f):
				return f, nil
			}
		case <-timer.C:
			return frame{}, errStepTimeout
		case <-s.ctx.Done():
			return frame{}, s.ctx.Err()
		}
	}
}

func isBinary(f frame) bool { return f.op == opBinary }

// send writes one frame. A failed write is explained by the read error that usually caused it (a
// reset stream or connection), when one arrives shortly; frames queued before that error are
// skipped, since the flow fails here either way.
func (s *session) send(op byte, payload []byte) error {
	err := s.ws.writeFrame(op, payload)
	if err == nil {
		return nil
	}
	if _, readErr := s.recv(time.Now().Add(pendingReadErrorWait), func(frame) bool { return false }); readErr != nil &&
		readErr != errStepTimeout && s.ctx.Err() == nil {
		return readErr
	}
	err = withCause(s.ws, err)
	if h2err := classifyHTTP2(err, netxlite.WriteOperation); h2err != nil {
		return h2err
	}
	return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
}

// awaitAddress waits for the server's REQUEST PUT /v1/address and returns it and its address.
// The first binary message must be that request.
func (s *session) awaitAddress() (*signalpb.WebSocketRequestMessage, string, error) {
	f, err := s.recv(time.Now().Add(s.flow.addressTimeout), isBinary)
	if err != nil {
		return nil, "", err
	}
	msg, err := parseMessage(f.payload)
	if err != nil {
		return nil, "", malformed(err)
	}
	if req := msg.GetRequest(); req.GetVerb() != "PUT" || req.GetPath() != "/v1/address" {
		return nil, "", errUnexpectedMessage
	}
	addr, err := provisioningAddress(msg.GetRequest().GetBody())
	if err != nil {
		return nil, "", malformed(err)
	}
	return msg.Request, addr, nil
}

// idle stays quiet for d: it answers pings and ignores other messages, and fails only on a
// close, a read error or the session's end.
func (s *session) idle(d time.Duration) error {
	if _, err := s.recv(time.Now().Add(d), func(frame) bool { return false }); err != errStepTimeout {
		return err
	}
	return nil
}

// keepalive sends GET /v1/keepalive, with a random body of pad bytes when pad > 0, and waits for
// a response (the server may also send requests, which are skipped).
func (s *session) keepalive(i uint64, pad int, zero time.Time) (*KeepaliveResult, error) {
	kr := &KeepaliveResult{Padding: pad, T0: time.Since(zero).Seconds()}
	fail := func(err error) (*KeepaliveResult, error) {
		kr.T, kr.Failure = time.Since(zero).Seconds(), measurexlite.NewFailure(err)
		return kr, err
	}
	id := uint64(time.Now().UnixMilli())*100 + i // #nosec G115 -- the apps use millisecond timestamps
	r := &signalpb.WebSocketRequestMessage{Verb: proto.String("GET"), Path: proto.String("/v1/keepalive"), Id: proto.Uint64(id)}
	if pad > 0 {
		r.Body = make([]byte, pad)
		_, _ = rand.Read(r.Body)
	}
	if err := s.send(opBinary, requestMessage(r)); err != nil {
		return fail(err)
	}
	var resp *signalpb.WebSocketResponseMessage
	_, err := s.recv(time.Now().Add(s.flow.keepaliveTimeout), func(f frame) bool {
		if f.op != opBinary {
			return false
		}
		if msg, err := parseMessage(f.payload); err == nil && msg.Response != nil {
			resp = msg.Response
		}
		return resp != nil
	})
	if err != nil {
		return fail(err)
	}
	kr.T = time.Since(zero).Seconds()
	kr.Status, kr.IDMatch = int64(resp.GetStatus()), resp.GetId() == id
	for _, h := range resp.GetHeaders() {
		if name, _, ok := strings.Cut(h, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "X-Signal-Timestamp") {
			kr.SignalTimestamp = true
		}
	}
	return kr, nil
}

// ping sends a PING with a random payload and reports whether a PONG echoed it. Unsolicited pongs
// (RFC 6455 allows them) are skipped; the result is false only if pongs arrived but none echoed
// the payload, and a timeout if none arrived.
func (s *session) ping() (bool, error) {
	payload := make([]byte, maxControlPayload)
	_, _ = rand.Read(payload)
	if err := s.send(opPing, payload); err != nil {
		return false, err
	}
	sawOtherPong := false
	_, err := s.recv(time.Now().Add(s.flow.keepaliveTimeout), func(f frame) bool {
		if f.op == opPong && !bytes.Equal(f.payload, payload) {
			sawOtherPong = true
			return false
		}
		return f.op == opPong
	})
	switch {
	case err == nil:
		return true, nil
	case err == errStepTimeout && sawOtherPong:
		return false, nil
	}
	return false, err
}
