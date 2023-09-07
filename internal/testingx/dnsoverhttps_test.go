package testingx

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/netem"
)

func TestDNSOverHTTPSHandler(t *testing.T) {
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
		name           string
		newHandler     func() http.Handler
		query          []byte
		expectStatus   int
		expectResponse []byte
	}

	testcases := []testconfig{{
		name: "when querying for an existing domain",
		newHandler: func() http.Handler {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return &DNSOverHTTPSHandler{
				RoundTripper: NewDNSRoundTripperWithDNSConfig(config),
			}
		},
		query:        exampleComQuery,
		expectStatus: 200,
		expectResponse: []byte{
			0x00, 0x01, // Transaction ID
			0x80, 0x00, // Flags (response, recursion desired)
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
		newHandler: func() http.Handler {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return &DNSOverHTTPSHandler{
				RoundTripper: NewDNSRoundTripperWithDNSConfig(config),
			}
		},
		query:        exampleOrgQuery,
		expectStatus: 200,
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
		newHandler: func() http.Handler {
			config := netem.NewDNSConfig()
			config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
			return &DNSOverHTTPSHandler{
				RoundTripper: NewDNSRoundTripperWithDNSConfig(config),
			}
		},
		query:          []byte{0x22},
		expectStatus:   500,
		expectResponse: []byte{},
	}, {
		name: "with internal round trip error",
		newHandler: func() http.Handler {
			return &DNSOverHTTPSHandler{
				RoundTripper: NewDNSRoundTripperSimulateTimeout(time.Millisecond, errors.New("antani")),
			}
		},
		query:          exampleComQuery,
		expectStatus:   500,
		expectResponse: []byte{},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.newHandler())
			defer server.Close()

			req, err := http.NewRequest("POST", server.URL, bytes.NewReader(tc.query))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Add("content-type", "application/dns-message")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.expectStatus {
				t.Fatal("invalid status code: expected", tc.expectStatus, "got", resp.StatusCode)
			}
			if resp.StatusCode != 200 {
				return
			}

			rawResponse, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

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
}
