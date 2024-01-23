package torx

//
// controlconn.go - code to manage the control connection.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// The various control port StatusCode constants.
const (
	ControlStatusOk            = 250
	ControlStatusOkUnnecessary = 251

	ControlStatusErrResourceExhausted      = 451
	ControlStatusErrSyntaxError            = 500
	ControlStatusErrUnrecognizedCmd        = 510
	ControlStatusErrUnimplementedCmd       = 511
	ControlStatusErrSyntaxErrorArg         = 512
	ControlStatusErrUnrecognizedCmdArg     = 513
	ControlStatusErrAuthenticationRequired = 514
	ControlStatusErrBadAuthentication      = 515
	ControlStatusErrUnspecifiedTorError    = 550
	ControlStatusErrInternalError          = 551
	ControlStatusErrUnrecognizedEntity     = 552
	ControlStatusErrInvalidConfigValue     = 553
	ControlStatusErrInvalidDescriptor      = 554
	ControlStatusErrUnmanagedEntity        = 555

	ControlStatusAsyncEvent = 650
)

// ControlTransport is a view of [*ControlConn] where we are
// only allowed to perform sync and async I/O actions.
type ControlTransport interface {
	// Notifications returns the channel from which one could read
	// the asynchronous events emitted by tor.
	Notifications() <-chan *ControlResponse

	// SendRecv sends a sync request and returns the corresponding
	// response returned by tor or an error.
	SendRecv(ctx context.Context, format string, args ...any) (*ControlResponse, error)
}

var _ ControlTransport = &ControlConn{}

// ControlConn is a tor control connection.
type ControlConn struct {
	// conn is the underlying [*textproto.Conn].
	conn *textproto.Conn

	// eof is used to signal the background workers
	// that it's now time to stop running.
	eof chan any

	// errRead contains the error that caused the read loop to exit.
	errRead error

	// errWrite contains the error that caused the write loop to exit.
	errWrite error

	// logger is the logger to use.
	logger model.Logger

	// notifications is the buffered channel where the read loop
	// posts notifications as soon as they arrive.
	notifications chan *ControlResponse

	// once provides once semantics for close.
	once *sync.Once

	// requests is the channel from which the write loop reads request.
	requests chan *controlRequest

	// waiters is the channel containing response waiters.
	waiters chan *controlResponseWaiter

	// wg tracks running goroutines.
	wg *sync.WaitGroup
}

// NewControlConn creates a new [*ControlConn] given a [io.ReadWriteCloser] and a [model.Logger].
func NewControlConn(conn io.ReadWriteCloser, logger model.Logger) *ControlConn {
	// initialize the conn
	const notificationsBuffer = 128
	c := &ControlConn{
		conn:          textproto.NewConn(conn),
		eof:           make(chan any),
		errRead:       nil,
		errWrite:      nil,
		logger:        logger,
		notifications: make(chan *ControlResponse, notificationsBuffer),
		once:          &sync.Once{},
		requests:      make(chan *controlRequest),
		waiters:       make(chan *controlResponseWaiter),
		wg:            &sync.WaitGroup{},
	}

	// run I/O loops in the background
	c.wg.Add(2)
	go c.readloop()
	go c.writeloop()

	// return to the caller
	return c
}

// Close closes the control connection.
func (c *ControlConn) Close() (err error) {
	c.once.Do(func() {
		c.logger.Debug("torx: control conn: close: start")

		// close the underlying conn to interrupt I/O.
		_ = c.conn.Close()

		// unblock channel readers and writers.
		close(c.eof)

		// wait for the background goroutines to stop running.
		c.wg.Wait()

		c.logger.Debug("torx: control conn: close: done")

		// compute the error to return giving
		// precedence to read errors
		//
		// note that we don't need to synchronize
		// access because of c.wg.Wait()
		switch {
		case c.errRead != nil:
			err = c.errRead
		case c.errWrite != nil:
			err = c.errWrite
		}
	})
	return
}

// SendRecv sends a request and receives the corresponding response.
func (c *ControlConn) SendRecv(ctx context.Context, format string, args ...any) (*ControlResponse, error) {
	// prepare the request
	req := newControlRequest(format, args...)

	// attempt to schedule it
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case c.requests <- req:
		// fallthrough
	}

	// await for the response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case resp := <-req.waiter.ch:
		return resp, nil
	}
}

// Notifications returns the channel receiving async events.
func (c *ControlConn) Notifications() <-chan *ControlResponse {
	return c.notifications
}

// readloop is the goroutine reading the control channel.
func (c *ControlConn) readloop() {
	defer func() {
		// tell the parent we stopped reading
		c.wg.Done()

		// close the conn (idempotent)
		_ = c.Close()

		c.logger.Debug("torx: control conn: readloop: done")
	}()

	c.logger.Debug("torx: control conn: readloop: start")
	for {
		// read the next response
		//
		// note: we POSSIBLY BLOCK when reading from the socket.
		resp, err := c.readResponse()
		if err != nil {
			c.errRead = err
			return
		}

		// handle notifications
		//
		// note: we use a buffered channel for dispatching
		// notification events to whoever cares
		if resp.Status == ControlStatusAsyncEvent {
			select {
			case c.notifications <- resp:
			default:
				// whatever
			}
			continue
		}

		// check whether we have someone awaiting
		// for a synchronous response
		//
		// note: the waiter channel is buffered
		// so we're not blocking
		select {
		case waiter := <-c.waiters:
			waiter.ch <- resp
		default:
			// nothing
		}
	}
}

// controlResponseWaiter wraps the channel waiting for a control response.
type controlResponseWaiter struct {
	ch chan *ControlResponse
}

// ErrControlConnTruncatedResponse indicates that a control response line was truncated.
var ErrControlConnTruncatedResponse = errors.New("torx: control conn: truncated response")

// ErrControlConnInvalidStatusCode indicates that the control response line status code is invalid.
var ErrControlConnInvalidStatusCode = errors.New("torx: control conn: invalid status code")

// ErrControlConnStatusCodeChanged indicates that subsequent lines in a control
// response have different status codes, which SHOULD NOT happen.
var ErrControlConnStatusCodeChanged = errors.New("torx: control conn: status code changed")

// ErrControlConnInvalidSeparator indicates we encountered an invalid separator
// when processing a tor control reponse.
var ErrControlConnInvalidSeparator = errors.New("torx: control conn: invalid separator")

// ControlResponse is a tor control response.
type ControlResponse struct {
	// Status is the status code shared by all the lines.
	Status int

	// Data contains the bytes read from MidReplyLine lines as well as the
	// bytes read from DataReplyLine lines.
	//
	// Each DataReplyLine is a single string containing all the content
	// encoded using the dot encoding and sent as a single unit.
	Data []string

	// EndReplyLine is the text in the final reply line.
	EndReplyLine string
}

// readResponse reads a control response from [*Conn].
func (c *ControlConn) readResponse() (resp *ControlResponse, err error) {
	for {
		// get the next protocol line
		//
		// note: we POSSIBLY BLOCK when reading from the socket.
		line, status, err := c.readResponseLineAndValidateStatusCode()
		if err != nil {
			return nil, err
		}

		// either initialize response or check consistency
		// in status code, which should not change since notifications
		// cannot be interlieved with response lines.
		switch {
		case resp == nil:
			resp = &ControlResponse{
				Status:       status,
				Data:         []string{}, // set later
				EndReplyLine: "",         // ditto
			}

		case resp.Status != status:
			return nil, ErrControlConnStatusCodeChanged
		}

		// check for the separator
		switch line[3] {
		case ' ':
			// final response line
			resp.EndReplyLine = line[4:]
			return resp, nil

		case '-':
			// continuation
			resp.Data = append(resp.Data, line[4:])
			continue

		case '+':
			// "dot-encoded" body
			dotBody, err := c.conn.ReadDotBytes()
			if err != nil {
				return nil, err
			}
			data := append([]byte{}, line[4:]...)
			data = append(data, dotBody...)
			resp.Data = append(resp.Data, string(data))

			// implementation note: cretz/bine removed tailing \r\n
			// from the response, which I think isn't needed.

			// note: we have already logged the first line
			// so here we only need to log the rest
			c.logger.Debugf("%v", string(dotBody))
			c.logger.Debugf("torx: control conn: < .")

		default:
			return nil, ErrControlConnInvalidSeparator
		}
	}
}

// readResponseLineAndValidateStatusCode reads a response line and validates the status code.
func (c *ControlConn) readResponseLineAndValidateStatusCode() (string, int, error) {
	// read the next line from the stream.
	//
	// note: we POSSIBLY BLOCK when reading from the socket.
	line, err := c.conn.ReadLine()
	if err != nil {
		return "", 0, err
	}

	c.logger.Debugf("torx: control conn: < %s", line)

	// we need four bytes for '<code:3><separator:1>'.
	if len(line) < 4 {
		return "", 0, ErrControlConnTruncatedResponse
	}

	// obtain the status status and make sure it is valid.
	status, err := strconv.Atoi(line[0:3])
	if err != nil || status < 100 || status > 900 {
		return "", 0, ErrControlConnInvalidStatusCode
	}

	return line, status, nil
}

// writeloop is the goroutine writing the control channel.
func (c *ControlConn) writeloop() {
	defer func() {
		// tell the parent we stopped writing
		c.wg.Done()

		// close the conn (idempotent)
		_ = c.Close()

		c.logger.Debug("torx: control conn: writeloop: done")
	}()

	c.logger.Debug("torx: control conn: writeloop: start")
	for {
		select {
		case <-c.eof:
			return

		case req := <-c.requests:
			c.logger.Debugf("torx: control conn: > %s", fmt.Sprintf(req.format, req.args...))

			// send request to tor
			//
			// note: we POSSIBLY BLOCK here when sending though it's
			// unlikely that we end up hading a full buffer.
			if err := c.conn.PrintfLine(req.format, req.args...); err != nil {
				c.errWrite = err
				return
			}

			// register the waiter
			//
			// note: we POSSIBLY BLOCK here if the reader is not draining
			// waiters because it's blocked reading.
			select {
			case c.waiters <- req.waiter:
			case <-c.eof:
				return
			}
		}
	}
}

// controlRequest is a request for the tor control port.
type controlRequest struct {
	// format is the format string.
	format string

	// args contains the args to format the format string.
	args []any

	// waiter is the corresponding waiter.
	waiter *controlResponseWaiter
}

// newControlRequest creates a new request.
func newControlRequest(format string, args ...any) *controlRequest {
	return &controlRequest{
		format: format,
		args:   args,
		waiter: &controlResponseWaiter{
			ch: make(chan *ControlResponse, 1), // avoid blocking readloop!
		},
	}
}
