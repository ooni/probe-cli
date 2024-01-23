package torcontrolnet

//
// conn.go - public implementation of the tor control conn.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"io"
	"net/textproto"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Conn is a tor control connection.
type Conn struct {
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
	notifications chan *Response

	// once provides once semantics for close.
	once *sync.Once

	// requests is the channel from which the write loop reads request.
	requests chan *request

	// waiters is the channel containing response waiters.
	waiters chan *responseWaiter

	// wg tracks running goroutines.
	wg *sync.WaitGroup
}

// NewConn creates a new [*Conn] given a [io.ReadWriteCloser] and a [model.Logger].
func NewConn(conn io.ReadWriteCloser, logger model.Logger) *Conn {
	// initialize the conn
	const notificationsBuffer = 128
	c := &Conn{
		conn:          textproto.NewConn(conn),
		eof:           make(chan any),
		errRead:       nil,
		errWrite:      nil,
		logger:        logger,
		notifications: make(chan *Response, notificationsBuffer),
		once:          &sync.Once{},
		requests:      make(chan *request),
		waiters:       make(chan *responseWaiter),
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
func (c *Conn) Close() (err error) {
	c.once.Do(func() {
		c.logger.Debug("torcontrol: close: start")

		// close the underlying conn to interrupt I/O.
		_ = c.conn.Close()

		// unblock channel readers and writers.
		close(c.eof)

		// wait for the background goroutines to stop running.
		c.wg.Wait()

		c.logger.Debug("torcontrol: close: done")

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
func (c *Conn) SendRecv(ctx context.Context, format string, args ...any) (*Response, error) {
	// prepare the request
	req := newRequest(format, args...)

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
func (c *Conn) Notifications() <-chan *Response {
	return c.notifications
}
