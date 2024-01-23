package torcontrol

//
// readloop.go - loop that reads control messages.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"bytes"
	"errors"
	"strconv"
)

// readloop is the goroutine reading the control channel.
func (c *Conn) readloop() {
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
		if resp.Status == StatusAsyncEvent {
			c.dispatchEvent(resp)
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

// responseWaiter wraps the channel waiting for a control response.
type responseWaiter struct {
	ch chan *Response
}

// ErrTruncatedResponse indicates that a control response line was truncated.
var ErrTruncatedResponse = errors.New("torcontrol: truncated response")

// ErrInvalidStatusCode indicates that the control response line status code is invalid.
var ErrInvalidStatusCode = errors.New("torcontrol: invalid status code")

// ErrStatusCodeChanged indicates that subsequent lines in a control
// response have different status codes, which SHOULD NOT happen.
var ErrStatusCodeChanged = errors.New("torcontrol: status code changed")

// ErrInvalidSeparator indicates that we encountered an invalid separator
// when processing a tor control reponse.
var ErrInvalidSeparator = errors.New("torcontrol: invalid separator")

// Response is a tor control response.
type Response struct {
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
func (c *Conn) readResponse() (resp *Response, err error) {
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
			resp = &Response{
				Status:       status,
				Data:         []string{}, // set later
				EndReplyLine: "",         // ditto
			}

		case resp.Status != status:
			return nil, ErrStatusCodeChanged
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
			data = bytes.TrimRight(data, "\r\n") // remove trailing newlines
			resp.Data = append(resp.Data, string(data))

			// note: we have already logged the first line
			// so here we only need to log the rest
			c.logger.Debugf("%v", string(dotBody))
			c.logger.Debugf("torcontrol: < .")

		default:
			return nil, ErrInvalidSeparator
		}
	}
}

// readResponseLineAndValidateStatusCode reads a response line and validates the status code.
func (c *Conn) readResponseLineAndValidateStatusCode() (string, int, error) {
	// read the next line from the stream.
	//
	// note: we POSSIBLY BLOCK when reading from the socket.
	line, err := c.conn.ReadLine()
	if err != nil {
		return "", 0, err
	}

	c.logger.Debugf("torcontrol: < %s", line)

	// we need four bytes for '<code:3><separator:1>'.
	if len(line) < 4 {
		return "", 0, ErrTruncatedResponse
	}

	// obtain the status status and make sure it is valid.
	status, err := strconv.Atoi(line[0:3])
	if err != nil || status < 100 || status > 900 {
		return "", 0, ErrInvalidStatusCode
	}

	return line, status, nil
}
