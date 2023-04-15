// Package stunx implements STUN and extends github.com/pion/stun.
//
// See https://www.rfc-editor.org/rfc/rfc8489.
package stunx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/pion/stun"
)

// ErrTransactionFailed is returned when the result is not BindingSuccess
var ErrTransactionFailed = errors.New("stunx: transaction failed")

// ErrMissingXORMappedAddressExtension is returned when the XOR-Mapped-Address extension is missing.
var ErrMissingXORMappedAddressExtension = errors.New("stunx: missing XOR-Mapped-Address extension")

// Client is a STUN client. The zero value of this struct is invalid; please,
// fill all the fields marked as MANDATORY or use [NewClient].
type Client struct {
	// CountDecodeErrors OPTIONALLY counts the number of decode errors.
	CountDecodeErrors *atomic.Int64

	// CountUnexpectedTransactionIDs OPTIONALLY counts the number of valid messages
	// received where the transaction ID is not the expected one.
	CountUnexpectedTransactionIDs *atomic.Int64

	// Endpoint is the MANDATORY UDP endpoint to use. The endpoint MUST contain an IP
	// address (quoted with [ and ] if IPv6) followed by a ":" and a port.
	//
	// For example: 1.2.3.4:5678, [::1:2:3:4]:5678.
	Endpoint string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MaxRequests is the MANDATORY maximum number of requests to send before
	// concluding that the STUN transaction timed out. RFC8489 Sect 6.2.1.
	// says that the default value for this field should be 7.
	MaxRequests int

	// MaxWaitTime is the MANDATORY maximum number of RTOs that the client
	// should wait before considering the transaction as timed out. According
	// to RFC8489 Sect 6.2.1., the default should be 16.
	MaxWaitTime int

	// ObserveTimeout is the OPTIONAL factory allowing tests to observe the
	// timeouts configured at each iteration of a transaction.
	ObserveTimeout func(timeo time.Duration)

	// RTO is the MANDATORY retransmission timeout. RFC8489 Sect 6.2.1.
	// says it SHOULD be >= than 500 ms.
	RTO time.Duration

	// TimeNow is MANDATORY and allows to mock obtaining the current time
	TimeNow func() time.Time

	// WrapConn is the OPTIONAL factory for wrapping the [net.Conn] connections
	// created by a [Client] during a STUN transaction.
	WrapConn func(conn net.Conn) net.Conn
}

// NewClient creates a new [Client] instance with the given endpoint and logger.
func NewClient(endpoint string, logger model.Logger) *Client {
	return &Client{
		CountDecodeErrors:             nil,
		CountUnexpectedTransactionIDs: nil,
		Endpoint:                      endpoint,
		Logger:                        logger,
		MaxRequests:                   7,
		MaxWaitTime:                   16,
		ObserveTimeout:                nil,
		RTO:                           500 * time.Millisecond,
		TimeNow:                       time.Now,
		WrapConn:                      nil,
	}
}

// LookupIPAddr sends a STUN binding request to the [Client] endpoint and waits
// for the corresponding response, managing timeouts and retransmissions.
//
// If the context deadline expires or the context is cancelled, the code will
// abort the pending transaction and return the context's error.
func (c *Client) LookupIPAddr(ctx context.Context) (string, error) {
	// create a dialer - must be without resolver so the caller is forced
	// to carefully choose which IP addresses to use
	dialer := netxlite.NewDialerWithoutResolver(c.Logger)

	// create the UDP connection
	conn, err := dialer.DialContext(ctx, "udp", c.Endpoint)
	if err != nil {
		return "", err
	}

	// allow the user to wrap the connection
	if c.WrapConn != nil {
		conn = c.WrapConn(conn)
	}

	// create the binding request message
	//
	// RFC8489 says "Resends of the same request reuse the same transaction ID", so
	// we create the request once and outside of the loop
	req := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// create channel for receiving the result IP address
	addrch := make(chan string, 1)

	// create channel for receiving the error
	errch := make(chan error, 1)

	// create channel for shutting down the sender
	donech := make(chan any)

	// create waitgroup to ensure we don't leak goroutines
	wg := &sync.WaitGroup{}

	// save the time when we started
	t0 := c.TimeNow()

	// start the sender goroutine in the background
	wg.Add(1)
	go c.senderLoop(wg, conn, req, donech, t0)

	// start the receiver goroutine in the background
	wg.Add(1)
	go c.receiverLoop(wg, conn, req, addrch, errch, t0)

	// await for events to occur and compute the final result.
	addr, err := func() (string, error) {
		select {
		case addr := <-addrch:
			return addr, nil

		case err := <-errch:
			return "", err

		case <-ctx.Done():
			return "", ctx.Err()
		}
	}()

	// signal the receiver it needs to stop.
	conn.Close()

	// signal the sender it needs to stop.
	close(donech)

	// block until the background goroutines have joined.
	wg.Wait()

	// finally, return result to the caller.
	return addr, err
}

// senderLoop is the loop that transmits (and retransmits) the request.
func (c *Client) senderLoop(
	wg *sync.WaitGroup,
	conn net.Conn,
	req *stun.Message,
	done <-chan any,
	t0 time.Time,
) {
	// synchronize with parent goroutine
	defer wg.Done()

	// track the number of attempts
	attempt := 0

	// set the initial timeout
	timeout := c.RTO

	// allow tests to observe the timeouts we're using
	if c.ObserveTimeout != nil {
		c.ObserveTimeout(timeout)
	}

	// arm the timer to fire after a RTO
	ticker := time.NewTicker(timeout)

	// loop waiting for rexmit events or completion
	for {
		// send the request to the server
		c.tdebugf(t0, "sending message: %s", req.String())
		_, _ = req.WriteTo(conn)

		// increment the number of attempts
		attempt++

		// make sure we eventually stop retrying
		if attempt >= c.MaxRequests {
			//	"If, after the last request, a duration equal to Rm times the RTO
			//	has passed without a response (providing ample time to get a response
			//	if only this final request actually succeeds), the client SHOULD
			//	consider the transaction to have failed. Rm SHOULD be configurable
			//	and SHOULD have a default of 16."
			//
			//		-- RFC 8489 Sect 6.2.1
			finalTimeout := time.Duration(c.MaxWaitTime) * c.RTO

			// allow tests to observe the timeouts we're using
			if c.ObserveTimeout != nil {
				c.ObserveTimeout(finalTimeout)
			}

			// set the read deadline and eventually stop the sender
			c.tdebugf(t0, "stopping the sender; setting read timeout to %v", finalTimeout)
			deadline := c.TimeNow().Add(finalTimeout)
			_ = conn.SetReadDeadline(deadline)
			return
		}

		select {
		case <-ticker.C:
			// double the timeout for the next iteration
			// see RFC8489 Sect 6.2.1.
			timeout *= 2

			// allow tests to observe the timeouts we're using
			if c.ObserveTimeout != nil {
				c.ObserveTimeout(timeout)
			}

			// rearm the rexmit timer
			ticker.Reset(timeout)

		case <-done:
			// the parent wants us to terminate ~now
			return
		}
	}
}

// receiverLoop is the loop that awaits for responses.
func (c *Client) receiverLoop(
	wg *sync.WaitGroup,
	conn net.Conn,
	req *stun.Message,
	ipAddrCh chan<- string,
	errCh chan<- error,
	t0 time.Time,
) {
	// synchronize with parent goroutine
	defer wg.Done()

	for {
		// create buffer for the response
		buffer := make([]byte, 1024)

		// receive a datagram from the network
		count, err := conn.Read(buffer)

		// the error is either the deadline exceeded or an ICMP error and
		// in both cases we consider the transaction failed
		//
		// because we're using netxlite under the hood, all the errors
		// have already been wrapped
		if err != nil {
			c.tdebugf(t0, "recv error: %s", err.Error())
			errCh <- err
			return
		}

		// prepare the response message to parse
		resp := &stun.Message{
			Raw: buffer[:count],
		}

		// parse the response message
		if err := resp.Decode(); err != nil {
			c.tdebugf(t0, "cannot decode message: %s", err.Error())
			if c.CountDecodeErrors != nil {
				c.CountDecodeErrors.Add(1)
			}
			continue
		}
		c.tdebugf(t0, "got message: %s", resp.String())

		// make sure the transaction ID is the expected one
		// see RFC8489 Sect 6.3
		if req.TransactionID != resp.TransactionID {
			c.tdebugf(t0, "unexpected transaction ID")
			if c.CountUnexpectedTransactionIDs != nil {
				c.CountUnexpectedTransactionIDs.Add(1)
			}
			continue
		}

		// if the response is not successful, we consider the transaction failed
		// see RFC8489 Sect 6.3.4
		if resp.Type != stun.BindingSuccess {
			errCh <- ErrTransactionFailed
			return
		}

		// unmarshal XOR-Mapped-Address extension
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(resp); err != nil {
			errCh <- ErrMissingXORMappedAddressExtension
			return
		}

		// return the IP address
		ipAddrCh <- xorAddr.IP.String()
		return
	}
}

// tdebugf is an utility function used to emit debug messages with timing information
func (c *Client) tdebugf(t0 time.Time, format string, v ...any) {
	elapsed := c.TimeNow().Sub(t0)
	c.Logger.Debugf("stunx: elapsed=%v - %s", elapsed, fmt.Sprintf(format, v...))
}
