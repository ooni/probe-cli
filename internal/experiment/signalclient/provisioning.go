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
	"errors"
	"io"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

// Timeouts for each step (variables so tests can shorten them).
var (
	addressTimeout   = 10 * time.Second // ProvisioningSocket.kt
	keepaliveTimeout = 10 * time.Second
)

// errTimeout is reported when a step does not complete in time (classified as generic_timeout_error).
var errTimeout = context.DeadlineExceeded

// errUnexpectedMessage means the server sent something the flow does not allow.
var errUnexpectedMessage = newFailure(FailureUnexpectedMessage, netxlite.ReadOperation, errors.New("signal_client: unexpected message"))

// malformed records a protobuf parse error with its failure string.
func malformed(err error) error {
	return newFailure(FailureMalformedMessage, netxlite.ReadOperation, err)
}

type frame struct {
	op      byte
	payload []byte
	err     error
}

// withCause replaces an error that only says our side is closed by the network error that caused
// the close, when ws knows it.
func withCause(ws *wsConn, err error) error {
	if ws.cause != nil && isLocalCloseError(err) {
		if cause := ws.cause(); cause != nil && !isLocalCloseError(cause) {
			return cause
		}
	}
	return err
}

// readFailure wraps an error from reading the WebSocket with its failure string.
func readFailure(err error) error {
	if errors.Is(err, errFrameTooLarge) || errors.Is(err, errInflate) {
		return malformed(err)
	}
	if h2err := classifyHTTP2(err, netxlite.ReadOperation); h2err != nil {
		return h2err
	}
	return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
}

// eofFailure is a WebSocket close from the server.
func eofFailure() error {
	return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, io.EOF)
}

// pendingReadErrorWait is how long a failed write waits for the read error that explains it.
const pendingReadErrorWait = 200 * time.Millisecond

// runProvisioning drives the flow on an upgraded WebSocket. paddings lists the keepalive body
// sizes; idle is how long to stay quiet after the ack (0: not at all); checkpoint, when not nil,
// is called after each step (the caller drains its trace's bounded event buffers there).
//
// Each step has one deadline, so frames the step ignores cannot extend it.
func runProvisioning(ctx context.Context, ws *wsConn, zero time.Time, paddings []int, idle time.Duration,
	checkpoint func()) *ProvisioningResult {
	res := &ProvisioningResult{IdleSeconds: int64(idle / time.Second)}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if checkpoint == nil {
		checkpoint = func() {}
	}

	frames := make(chan frame, 16)
	go func() {
		for {
			op, p, err := ws.readFrame()
			if err != nil {
				err = readFailure(withCause(ws, err))
			}
			select {
			case frames <- frame{op, p, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	// next returns the next data or pong frame before deadline, answering pings on the way.
	next := func(deadline time.Time) (frame, error) {
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		for {
			select {
			case f := <-frames:
				if f.err != nil {
					return f, f.err
				}
				switch f.op {
				case opPing:
					_ = ws.writeFrame(opPong, f.payload)
					continue
				case opClose:
					return f, eofFailure()
				}
				return f, nil
			case <-timer.C:
				return frame{}, errTimeout
			case <-ctx.Done():
				return frame{}, ctx.Err()
			}
		}
	}
	// writeFailure explains a failed write by the read error that usually caused it (a reset
	// stream or connection), when one arrives shortly; otherwise it reports the write error.
	writeFailure := func(err error) error {
		select {
		case f := <-frames:
			if f.err != nil {
				return f.err
			}
		case <-time.After(pendingReadErrorWait):
		}
		err = withCause(ws, err)
		if h2err := classifyHTTP2(err, netxlite.WriteOperation); h2err != nil {
			return h2err
		}
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	fail := func(step string, err error) *ProvisioningResult {
		res.FailedStep = step
		res.Failure = measurexlite.NewFailure(err)
		return res
	}

	// 1. the server's provisioning address
	var req *wsRequest
	deadline := time.Now().Add(addressTimeout)
	for req == nil {
		f, err := next(deadline)
		if err != nil {
			return fail("provisioning_address", err)
		}
		if f.op != opBinary {
			continue
		}
		msg, err := parseMessage(f.payload)
		if err != nil {
			return fail("provisioning_address", malformed(err))
		}
		if msg.Request == nil || msg.Request.Verb != "PUT" || msg.Request.Path != "/v1/address" {
			return fail("provisioning_address", errUnexpectedMessage)
		}
		req = msg.Request
	}
	addr, err := provisioningAddress(req.Body)
	if err != nil {
		return fail("provisioning_address", malformed(err))
	}
	res.AddressReceived, res.AddressLength, res.AddressT = true, len(addr), time.Since(zero).Seconds()

	// 2. ack, as the apps do
	ack := &wsResponse{ID: req.ID, Status: 200, Message: "OK"}
	if err := ws.writeFrame(opBinary, ack.marshal()); err != nil {
		return fail("ack", writeFailure(err))
	}
	res.Acked = true
	checkpoint()

	if idle > 0 {
		// stay quiet for the full idle time: answer pings, ignore other messages, stop only on
		// a close, an error or cancellation
		timer := time.NewTimer(idle)
		for idling := true; idling; {
			select {
			case f := <-frames:
				switch {
				case f.err != nil:
					timer.Stop()
					return fail("idle", f.err)
				case f.op == opClose:
					timer.Stop()
					return fail("idle", eofFailure())
				case f.op == opPing:
					_ = ws.writeFrame(opPong, f.payload)
				}
			case <-timer.C:
				idling = false
			case <-ctx.Done():
				timer.Stop()
				return fail("idle", ctx.Err())
			}
		}
		checkpoint()
	}

	// 3. keepalives, some padded
	for i, pad := range paddings {
		kr := &KeepaliveResult{Padding: pad}
		res.Keepalives = append(res.Keepalives, kr)
		id := uint64(time.Now().UnixMilli())*100 + uint64(i) // #nosec G115 -- the apps use millisecond timestamps
		r := &wsRequest{Verb: "GET", Path: "/v1/keepalive", ID: id}
		if pad > 0 {
			r.Body = make([]byte, pad)
			_, _ = rand.Read(r.Body)
		}
		kr.T0 = time.Since(zero).Seconds()
		keepaliveFail := func(err error) *ProvisioningResult {
			kr.T, kr.Failure = time.Since(zero).Seconds(), measurexlite.NewFailure(err)
			return fail("keepalive", err)
		}
		if err := ws.writeFrame(opBinary, r.marshal()); err != nil {
			return keepaliveFail(writeFailure(err))
		}
		deadline := time.Now().Add(keepaliveTimeout)
		for {
			f, err := next(deadline)
			if err != nil {
				return keepaliveFail(err)
			}
			if f.op != opBinary {
				continue
			}
			msg, err := parseMessage(f.payload)
			if err != nil || msg.Response == nil {
				continue // the server may also send requests; we only wait for our response
			}
			kr.T = time.Since(zero).Seconds()
			kr.Status, kr.IDMatch = int64(msg.Response.Status), msg.Response.ID == id
			for _, h := range msg.Response.Headers {
				if name, _, ok := strings.Cut(h, ":"); ok && strings.EqualFold(strings.TrimSpace(name), "X-Signal-Timestamp") {
					kr.SignalTimestamp = true
				}
			}
			break
		}
		checkpoint()
	}

	// 4. a PING whose payload a PONG must echo. Unsolicited pongs (RFC 6455 allows them) are
	// skipped; ping_echoed is false only if pongs arrived but none echoed the payload.
	payload := make([]byte, 125)
	_, _ = rand.Read(payload)
	if err := ws.writeFrame(opPing, payload); err != nil {
		return fail("ping", writeFailure(err))
	}
	sawOtherPong := false
	timer := time.NewTimer(keepaliveTimeout)
	defer timer.Stop()
	for res.PingEchoed == nil {
		select {
		case f := <-frames:
			switch {
			case f.err != nil:
				return fail("ping", f.err)
			case f.op == opClose:
				return fail("ping", eofFailure())
			case f.op == opPong && bytes.Equal(f.payload, payload):
				echoed := true
				res.PingEchoed = &echoed
			case f.op == opPong:
				sawOtherPong = true
			case f.op == opPing:
				_ = ws.writeFrame(opPong, f.payload)
			}
		case <-timer.C:
			if !sawOtherPong {
				return fail("ping", errTimeout)
			}
			echoed := false
			res.PingEchoed = &echoed
		case <-ctx.Done():
			return fail("ping", ctx.Err())
		}
	}
	checkpoint()

	_ = ws.writeFrame(opClose, []byte{0x03, 0xe8}) // 1000, normal closure; the flow already succeeded
	return res
}
