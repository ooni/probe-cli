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

// runProvisioning drives the flow on an upgraded WebSocket. paddings lists the keepalive body
// sizes; idle is how long to stay quiet after the ack (0: not at all).
func runProvisioning(ctx context.Context, ws *wsConn, zero time.Time, paddings []int, idle time.Duration) *ProvisioningResult {
	res := &ProvisioningResult{IdleSeconds: int64(idle / time.Second)}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	frames := make(chan frame, 16)
	go func() {
		for {
			op, p, err := ws.readFrame()
			if err != nil {
				err = netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
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

	// next returns the next data or pong frame, answering pings on the way.
	next := func(timeout time.Duration) (frame, error) {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case f := <-frames:
				if f.err != nil {
					return f, f.err
				}
				switch f.op {
				case opPing:
					ws.writeFrame(opPong, f.payload)
					continue
				case opClose:
					return f, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, io.EOF)
				}
				return f, nil
			case <-timer.C:
				return frame{}, errTimeout
			case <-ctx.Done():
				return frame{}, ctx.Err()
			}
		}
	}
	fail := func(step string, err error) *ProvisioningResult {
		res.FailedStep = step
		res.Failure = measurexlite.NewFailure(err)
		return res
	}

	// 1. the server's provisioning address
	var req *wsRequest
	for req == nil {
		f, err := next(addressTimeout)
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
		return fail("ack", netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err))
	}
	res.Acked = true

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
					return fail("idle", netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, io.EOF))
				case f.op == opPing:
					ws.writeFrame(opPong, f.payload)
				}
			case <-timer.C:
				idling = false
			case <-ctx.Done():
				timer.Stop()
				return fail("idle", ctx.Err())
			}
		}
	}

	// 3. keepalives, some padded
	for i, pad := range paddings {
		kr := &KeepaliveResult{Padding: pad}
		res.Keepalives = append(res.Keepalives, kr)
		id := uint64(time.Now().UnixMilli())*100 + uint64(i) // the apps use millisecond timestamps
		r := &wsRequest{Verb: "GET", Path: "/v1/keepalive", ID: id}
		if pad > 0 {
			r.Body = make([]byte, pad)
			rand.Read(r.Body)
		}
		kr.T0 = time.Since(zero).Seconds()
		if err := ws.writeFrame(opBinary, r.marshal()); err != nil {
			kr.Failure = measurexlite.NewFailure(netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err))
			return fail("keepalive", err)
		}
		for {
			f, err := next(keepaliveTimeout)
			if err != nil {
				kr.Failure = measurexlite.NewFailure(err)
				return fail("keepalive", err)
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
	}

	// 4. a PING whose payload the PONG must echo
	payload := make([]byte, 125)
	rand.Read(payload)
	if err := ws.writeFrame(opPing, payload); err != nil {
		return fail("ping", netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err))
	}
	timer := time.NewTimer(keepaliveTimeout)
	defer timer.Stop()
	for res.PingEchoed == nil {
		select {
		case f := <-frames:
			if f.err != nil {
				return fail("ping", f.err)
			}
			switch f.op {
			case opPong:
				echoed := bytes.Equal(f.payload, payload)
				res.PingEchoed = &echoed
			case opPing:
				ws.writeFrame(opPong, f.payload)
			}
		case <-timer.C:
			return fail("ping", errTimeout)
		}
	}

	ws.writeFrame(opClose, []byte{0x03, 0xe8}) // 1000, normal closure
	return res
}
