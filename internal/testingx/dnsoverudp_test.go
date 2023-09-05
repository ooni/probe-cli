package testingx

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestDNSOverUDPHandler(t *testing.T) {
	exampleComQuery := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Flags
		0x00, 0x01, // Questions
		0x00, 0x00, // Answer RRs
		0x00, 0x00, // Authority RRs
		0x00, 0x00, // Additional RRs
		// QNAME
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // Null-terminator of QNAME
		0x00, 0x01, // QTYPE (A record)
		0x00, 0x01, // QCLASS (IN)
	}

	exampleOrgQuery := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Flags
		0x00, 0x01, // Questions
		0x00, 0x00, // Answer RRs
		0x00, 0x00, // Authority RRs
		0x00, 0x00, // Additional RRs
		// QNAME
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'o', 'r', 'g',
		0x00,       // Null-terminator of QNAME
		0x00, 0x01, // QTYPE (A record)
		0x00, 0x01, // QCLASS (IN)
	}

	type testconfig struct {
		name            string
		newRoundTripper func() DNSRoundTripper
		query           []byte
		expectErr       error
		expectResponse  []byte
	}

	testcases := []testconfig{{
		name: "when querying for an existing domain",
		newRoundTripper: func() DNSRoundTripper {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return NewDNSRoundTripperWithDNSConfig(config)
		},
		query:     exampleComQuery,
		expectErr: nil,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x00, // Flags (response)
			0x00, 0x01, // Num questions
			0x00, 0x02, // Num asnwers RRs
			0x00, 0x00, // Num Authority RRs
			0x00, 0x00, // Num Additional RRs

			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN

			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN
			0x00, 0x00, 0x0e, 0x10, // TTL = 3600 seconds
			0x00, 0x04, // data length: 4 bytes
			0x5d, 0xb8, 0xd8, 0x22, // IPv4 address (93.184.216.34)

			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x05, // type = CNAME
			0x00, 0x01, // class = IN
			0x00, 0x00, 0x0e, 0x10, // TTL = 3600 seconds
			0x00, 0x13, // data length = 19 bytes
			0x05, 0x77, 0x65, 0x62, 0x30, 0x31, // QNAME: 5(web01)
			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00, // QNAME: null terminator
		},
	}, {
		name: "when querying for a nonexisting domain",
		newRoundTripper: func() DNSRoundTripper {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return NewDNSRoundTripperWithDNSConfig(config)
		},
		query:     exampleOrgQuery,
		expectErr: nil,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x03, // Flags (Response, NXDOMAIN)
			0x00, 0x01, // Num questions
			0x00, 0x00, // Num answers RRs
			0x00, 0x00, // Num authority RRs
			0x00, 0x00, // Num additional RRs
			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x6f, 0x72, 0x67, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN
		},
	}, {
		name: "with invalid query",
		newRoundTripper: func() DNSRoundTripper {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return NewDNSRoundTripperWithDNSConfig(config)
		},
		query:          []byte{0x22},
		expectErr:      os.ErrDeadlineExceeded,
		expectResponse: []byte{},
	}, {
		name: "with round trip timeout",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperSimulateTimeout(time.Millisecond, errors.New("antani"))
		},
		query:          exampleComQuery,
		expectErr:      os.ErrDeadlineExceeded,
		expectResponse: []byte{},
	}, {
		name: "with DNSRoundTripperEmptyResponse and valid query",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperEmptyRespnse()
		},
		query:     exampleComQuery,
		expectErr: nil,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x00, // Flags (response)
			0x00, 0x01, // Num questions
			0x00, 0x00, // Num asnwers RRs
			0x00, 0x00, // Num Authority RRs
			0x00, 0x00, // Num Additional RRs

			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN
		},
	}, {
		name: "with DNSRoundTripperEmptyResponse and and invalid query",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperEmptyRespnse()
		},
		query:          []byte{0x22},
		expectErr:      os.ErrDeadlineExceeded,
		expectResponse: []byte{},
	}, {
		name: "with DNSRoundTripperRefused and valid query",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperRefused()
		},
		query:     exampleComQuery,
		expectErr: nil,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x05, // Flags (response, refused)
			0x00, 0x01, // Num questions
			0x00, 0x00, // Num asnwers RRs
			0x00, 0x00, // Num Authority RRs
			0x00, 0x00, // Num Additional RRs

			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x63, 0x6f, 0x6d, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN
		},
	}, {
		name: "with DNSRoundTripperEmptyResponse and and invalid query",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperRefused()
		},
		query:          []byte{0x22},
		expectErr:      os.ErrDeadlineExceeded,
		expectResponse: []byte{},
	}, {
		name: "with DNSRoundTripperNXDOMAIN",
		newRoundTripper: func() DNSRoundTripper {
			return NewDNSRoundTripperNXDOMAIN()
		},
		query:     exampleOrgQuery,
		expectErr: nil,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x03, // Flags (Response, NXDOMAIN)
			0x00, 0x01, // Num questions
			0x00, 0x00, // Num answers RRs
			0x00, 0x00, // Num authority RRs
			0x00, 0x00, // Num additional RRs
			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65, // QNAME: 7(example)
			0x03, 0x6f, 0x72, 0x67, // QNAME: 3(com)
			0x00,       // QNAME: null terminator
			0x00, 0x01, // type = A
			0x00, 0x01, // class = IN
		},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			udpAddr := &net.UDPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: 0,
			}
			listener := MustNewDNSOverUDPListener(udpAddr, &DNSOverUDPListenerStdlib{}, tc.newRoundTripper())
			defer listener.Close()

			pconn, err := net.Dial("udp", listener.LocalAddr().String())
			if err != nil {
				t.Fatal(err)
			}
			pconn.SetDeadline(time.Now().Add(250 * time.Millisecond))
			_, _ = pconn.Write(tc.query)

			buffer := make([]byte, 1<<14)
			count, err := pconn.Read(buffer)

			switch {
			case tc.expectErr == nil && err != nil:
				t.Fatal("expected no error but got", err)
			case tc.expectErr != nil && err == nil:
				t.Fatal("expected", tc.expectErr, "but got", err)
			case tc.expectErr != nil && err != nil:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("expected", tc.expectErr, "but got", err)
				}
				return
			default:
				// fallthrough
			}

			if err != nil {
				t.Fatal(err)
			}

			rawResponse := buffer[:count]
			msg := &dns.Msg{}
			if err := msg.Unpack(rawResponse); err != nil {
				t.Fatal(err)
			}
			t.Logf("\n%s", msg)
			t.Logf("%#v", rawResponse)

			if diff := cmp.Diff(tc.expectResponse, rawResponse); diff != "" {
				t.Fatal(diff)
			}
		})
	}

	t.Run("when there is an error reading in the main loop", func(t *testing.T) {
		called := &atomic.Bool{}
		rtx := &DNSOverUDPListener{
			cancel: func() {
				// nothing to do here
			},
			closeOnce: sync.Once{},
			pconn: &mocks.UDPLikeConn{
				MockReadFrom: func(p []byte) (int, net.Addr, error) {
					if called.Load() {
						return 0, nil, net.ErrClosed
					}
					called.Store(true)
					return 0, nil, errors.New("mocked error")
				},
			},
			rtx: nil,
			wg:  sync.WaitGroup{},
		}

		rtx.wg.Add(1)
		go rtx.mainloop(context.Background())
		rtx.wg.Wait()
	})
}
