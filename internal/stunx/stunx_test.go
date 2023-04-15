package stunx

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/stuntesting"
	"github.com/pion/stun"
)

// timeoutObserver allows to observe the timeouts we configured.
type timeoutObserver struct {
	mu sync.Mutex
	t  []time.Duration
}

// ObserveTimeout observes a given timeout.
func (to *timeoutObserver) ObserveTimeout(t time.Duration) {
	defer to.mu.Unlock()
	to.mu.Lock()
	to.t = append(to.t, t)
}

// Timeouts returns a copy of the observed timeouts.
func (to *timeoutObserver) Timeouts() []time.Duration {
	defer to.mu.Unlock()
	to.mu.Lock()
	return append([]time.Duration{}, to.t...)
}

// This test ensures the client is WAI in normal operating conditions.
func TestClientWorkingAsIntended(t *testing.T) {
	// testcase is a test case for this test
	type testcase struct {
		// name is the MANDATORY test-case name
		name string

		// contextTimeout is the OPTIONAL timeout for the context
		contextTimeout time.Duration

		// handler is the MANDATORY handler for the [stuntesting.Server]
		handler stuntesting.Handler

		// modifyClient is an OPTIONAL hook to modify the client before using it
		modifyClient func(c *Client)

		// timeoutObserver is the OPTIONAL [timeoutObserver]
		timeoutObserver *timeoutObserver

		// expectErr is the MANDATORY expected error
		expectErr error

		// expectAddr is the MANDATORY expected address
		expectAddr string

		// checkTimeouts is the OPTIONAL hook to check for timeouts
		checkTimeouts func([]time.Duration) error

		// checkClient is the OPTIONAL hook to check client fields after the run
		checkClient func(c *Client) error
	}

	// testcases defines all the test cases
	testcases := []testcase{{
		name:            "when everything works as expected",
		contextTimeout:  0,
		handler:         stuntesting.ResponseWithAddPort(net.IPv4(8, 8, 4, 4), 443),
		modifyClient:    nil,
		timeoutObserver: nil,
		expectErr:       nil,
		expectAddr:      "8.8.4.4",
		checkTimeouts:   nil,
		checkClient:     nil,
	}, {
		name:           "when the transaction times out",
		contextTimeout: 0,
		handler: stuntesting.HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			// the server won't respond because we're returning an error here
			return nil, errors.New("mocked error")
		}),
		modifyClient: func(c *Client) {
			c.RTO = 5 * time.Millisecond // scaled down by a 100 factor
		},
		timeoutObserver: &timeoutObserver{},
		expectErr:       os.ErrDeadlineExceeded,
		expectAddr:      "",
		checkTimeouts: func(d []time.Duration) error {
			expect := []time.Duration{
				5 * time.Millisecond,
				10 * time.Millisecond,
				20 * time.Millisecond,
				40 * time.Millisecond,
				80 * time.Millisecond,
				160 * time.Millisecond,
				320 * time.Millisecond,
				80 * time.Millisecond, // final timeout for reading
			}
			if diff := cmp.Diff(expect, d); diff != "" {
				return errors.New(diff)
			}
			return nil
		},
		checkClient: nil,
	}, {
		name:           "when the context times out",
		contextTimeout: 200 * time.Millisecond,
		handler: stuntesting.HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			// the server won't respond because we're returning an error here
			return nil, errors.New("mocked error")
		}),
		modifyClient:    nil,
		timeoutObserver: nil,
		expectErr:       context.DeadlineExceeded,
		expectAddr:      "",
		checkTimeouts:   nil,
		checkClient:     nil,
	}, {
		name:           "with wrong transaction ID",
		contextTimeout: 0,
		handler: stuntesting.HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			resp := stun.MustBuild(stun.BindingSuccess, stun.TransactionID)
			for req.TransactionID == resp.TransactionID {
				resp.TransactionID = stun.NewTransactionID()
			}
			addr := &stun.XORMappedAddress{
				IP:   net.IPv4(8, 8, 4, 4),
				Port: 443,
			}
			runtimex.Try0(addr.AddTo(resp))
			return resp, nil

		}),
		modifyClient: func(c *Client) {
			c.RTO = 5 * time.Millisecond // scaled down by a 100 factor
			c.CountUnexpectedTransactionIDs = &atomic.Int64{}
		},
		timeoutObserver: nil,
		expectErr:       os.ErrDeadlineExceeded,
		expectAddr:      "",
		checkTimeouts:   nil,
		checkClient: func(c *Client) error {
			const expect = 7
			if n := c.CountUnexpectedTransactionIDs.Load(); n != expect {
				return fmt.Errorf("expected %d unexpected transaction; got %d", expect, n)
			}
			return nil
		},
	}, {
		name:           "when the response indicates an error",
		contextTimeout: 0,
		handler: stuntesting.HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			resp := stun.MustBuild(stun.BindingError)
			resp.TransactionID = req.TransactionID
			return resp, nil
		}),
		modifyClient:    nil,
		timeoutObserver: nil,
		expectErr:       ErrTransactionFailed,
		expectAddr:      "",
		checkTimeouts:   nil,
		checkClient:     nil,
	}, {
		name:           "when the XOR-Mapped-Address extension is missing",
		contextTimeout: 0,
		handler: stuntesting.HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			resp := stun.MustBuild(stun.BindingSuccess)
			resp.TransactionID = req.TransactionID
			return resp, nil
		}),
		modifyClient:    nil,
		timeoutObserver: nil,
		expectErr:       ErrMissingXORMappedAddressExtension,
		expectAddr:      "",
		checkTimeouts:   nil,
		checkClient:     nil,
	}}

	// run all the test cases
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// create the STUN server
			server := stuntesting.MustNewServer(tc.handler)
			defer server.Close()

			// create the client
			client := NewClient(server.Address(), model.DiscardLogger)

			// if there's a timeout observer, use it
			if tc.timeoutObserver != nil {
				client.ObserveTimeout = tc.timeoutObserver.ObserveTimeout
			}

			// optionally modify the client before using it
			if tc.modifyClient != nil {
				tc.modifyClient(client)
			}

			// create the request context
			ctx := context.Background()

			// if there's a context timeout, apply it
			if tc.contextTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.contextTimeout)
				defer cancel()
			}

			// issue the STUN binding request
			addr, err := client.LookupIPAddr(ctx)

			// check whether the error is the expected one
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}

			// check whether the address is the expected one
			if diff := cmp.Diff(tc.expectAddr, addr); diff != "" {
				t.Fatal(diff)
			}

			// optionally check the timeouts
			runtimex.Assert(
				(tc.timeoutObserver == nil && tc.checkTimeouts == nil) ||
					(tc.timeoutObserver != nil && tc.checkTimeouts != nil),
				"you must set both timeoutObserver and checkTimeouts",
			)
			if tc.timeoutObserver != nil && tc.checkTimeouts != nil {
				timeouts := tc.timeoutObserver.Timeouts()
				if err := tc.checkTimeouts(timeouts); err != nil {
					t.Fatal(err)
				}
			}

			// optionally check the client fields
			if tc.checkClient != nil {
				if err := tc.checkClient(client); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// This test ensures that dialing fails if the user provides a domain name
func TestClientDialWithDomainNameFails(t *testing.T) {
	client := NewClient("stun.l.google.com:19302", model.DiscardLogger)

	// attempt to obtain the IP address
	addr, err := client.LookupIPAddr(context.Background())

	// we expect a dialing error
	if !errors.Is(err, netxlite.ErrNoResolver) {
		t.Fatal("unexpected error", err)
	}

	// we expect an empty addr
	if addr != "" {
		t.Fatal("expected empty addr")
	}
}

// This test ensures that we can wrap the net.Conn used for communicating with the server.
func TestClientAllowsWrappingTheNetConn(t *testing.T) {
	// create the server
	server := stuntesting.MustNewServer(stuntesting.ResponseWithAddPort(
		net.IPv4(8, 8, 4, 4),
		443,
	))
	defer server.Close()

	// create the client
	client := NewClient(server.Address(), model.DiscardLogger)

	// create the bytecounter
	counter := bytecounter.New()

	// wrap the connection to count bytes
	client.WrapConn = func(conn net.Conn) net.Conn {
		return bytecounter.WrapConn(conn, counter)
	}

	// attempt to obtain the IP address
	addr, err := client.LookupIPAddr(context.Background())

	// we expect to see no errors
	if err != nil {
		t.Fatal(err)
	}

	// we expect a specific address
	if addr != "8.8.4.4" {
		t.Fatal("unexpected addr", addr)
	}

	// we expect to see bytes sent and received
	const expectedRecv = 32
	if n := counter.BytesReceived(); n != expectedRecv {
		t.Fatal("expected", expectedRecv, "bytes received; got", n)
	}
	const expectedSent = 20
	if n := counter.BytesSent(); n != expectedSent {
		t.Fatal("expected", expectedSent, "bytes sent; got", n)
	}
}

// This test verifies that the client handles the case where it is sent
// garbage data that is not encoded using the STUN data format
func TestClientCannotDecodeResponse(t *testing.T) {
	// create an UDP listener
	serverConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer serverConn.Close()

	// serve the connection by responding with garbage
	go func() {
		for {
			buffer := make([]byte, 1024)
			count, addr, err := serverConn.ReadFrom(buffer)
			if err != nil {
				return
			}
			_ = buffer[:count] // just ignore the request
			resp := []byte{0x00, 0x01, 0x02}
			_, _ = serverConn.WriteTo(resp, addr)
		}
	}()

	// obtain the listener endpoint
	endpoint := serverConn.LocalAddr().String()

	// create a client
	client := NewClient(endpoint, model.DiscardLogger)

	// make sure we count the number of decode errors
	client.CountDecodeErrors = &atomic.Int64{}

	// make sure this test doesn't run for a very long time by setting a
	// context deadline that is 1/10 of the default RTO
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// attempt to obtain the IP address
	addr, err := client.LookupIPAddr(ctx)

	// we expect the deadline to terminate the transaction
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("unexpected error", err)
	}

	// we expect an empty addr
	if addr != "" {
		t.Fatal("expected empty addr")
	}

	// we expect to see a single decode error.
	if client.CountDecodeErrors.Load() != 1 {
		t.Fatal("expected to see a decode error")
	}
}
