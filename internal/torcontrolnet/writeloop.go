package torcontrolnet

//
// writeloop.go - implementation of the write loop.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import "fmt"

// writeloop is the goroutine writing the control channel.
func (c *Conn) writeloop() {
	defer func() {
		// tell the parent we stopped writing
		c.wg.Done()

		// close the conn (idempotent)
		_ = c.Close()

		c.logger.Debug("torcontrol: writeloop: done")
	}()

	c.logger.Debug("torcontrol: writeloop: start")
	for {
		select {
		case <-c.eof:
			return

		case req := <-c.requests:
			c.logger.Debugf("torcontrol: > %s", fmt.Sprintf(req.format, req.args...))

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

// request is a request for the tor control port.
type request struct {
	// format is the format string.
	format string

	// args contains the args to format the format string.
	args []any

	// waiter is the corresponding waiter.
	waiter *responseWaiter
}

// newRequest creates a new request.
func newRequest(format string, args ...any) *request {
	return &request{
		format: format,
		args:   args,
		waiter: &responseWaiter{
			ch: make(chan *Response, 1), // avoid blocking readloop!
		},
	}
}
