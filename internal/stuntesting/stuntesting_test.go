package stuntesting

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/pion/stun"
)

func TestServer(t *testing.T) {

	// testcase is a test case for this test
	type testcase struct {
		// name is the test case name
		name string

		// handler is the [Handler] to install
		handler Handler

		// request contains the request to send
		request []byte

		// readTimeout contains the read timeout to configure
		readTimeout time.Duration

		// expectErr is the error we expect
		expectErr error

		// expectResp contains the expected expectResp
		expectResp []byte
	}

	// testcases contains all test cases
	testcases := []testcase{{
		name:    "for a valid request",
		handler: ResponseWithAddPort(net.IPv4(8, 8, 4, 4), 443),
		request: []byte{
			// message type: binding request
			0x00, 0x01,

			// message length: 0 byte
			0x00, 0x00,

			// cookie
			0x21, 0x12, 0xa4, 0x42,

			// transaction ID
			0xef, 0x58, 0x91, 0xe0, 0x84, 0xb6, 0x6a, 0x85, 0x7a, 0x6e, 0x15, 0x24,
		},
		readTimeout: time.Second,
		expectErr:   nil,
		expectResp: []byte{
			// message type: binding success response
			0x01, 0x01,

			// message length: 12
			0x00, 0x0c,

			// cookie
			0x21, 0x12, 0xa4, 0x42,

			// transaction ID
			0xef, 0x58, 0x91, 0xe0, 0x84, 0xb6, 0x6a, 0x85, 0x7a, 0x6e, 0x15, 0x24,

			// attribute type: XOR-Mapped-Address
			0x00, 0x20,

			// attribute length: 8
			0x00, 0x08,

			// reserved
			0x00,

			// protocol family: IPv4
			0x01,

			// port Xor-ed
			0x20, 0xa9,

			// IP addr Xor-ed
			0x29, 0x1a, 0xa0, 0x46,
		},
	}, {
		name:        "when decode fails",
		handler:     nil,
		request:     []byte{0x01, 0x02, 0x03, 0x04},
		readTimeout: 250 * time.Millisecond,
		expectErr:   os.ErrDeadlineExceeded,
		expectResp:  nil,
	}, {
		name: "when serve fails",
		handler: HandlerFunc(func(req *stun.Message) (*stun.Message, error) {
			return nil, errors.New("mocked error")
		}),
		request: []byte{
			// message type: binding request
			0x00, 0x01,

			// message length: 0 byte
			0x00, 0x00,

			// cookie
			0x21, 0x12, 0xa4, 0x42,

			// transaction ID
			0xef, 0x58, 0x91, 0xe0, 0x84, 0xb6, 0x6a, 0x85, 0x7a, 0x6e, 0x15, 0x24,
		},
		readTimeout: 250 * time.Millisecond,
		expectErr:   os.ErrDeadlineExceeded,
		expectResp:  nil,
	}}

	// run all the test cases
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// create the server
			server := MustNewServer(tc.handler)
			defer server.Close()

			// create client connection
			pconn := runtimex.Try1(net.Dial("udp", server.Address()))

			// send the request
			_ = runtimex.Try1(pconn.Write(tc.request))

			// set the read deadline
			_ = pconn.SetReadDeadline(time.Now().Add(tc.readTimeout))

			// read the response
			buffer := make([]byte, 1024)
			count, err := pconn.Read(buffer)

			// make sure we've got the expected error
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}

			// stop processing if we've got an error
			if err != nil {
				return
			}

			// compare the response with the expected response
			resp := buffer[:count]
			if diff := cmp.Diff(tc.expectResp, resp); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
