package testingx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

	config := netem.NewDNSConfig()
	config.AddRecord("example.com", "web01.example.com", "93.184.216.34")
	handler := &DNSOverHTTPSHandler{
		Config: config,
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	type testconfig struct {
		name           string
		query          []byte
		expectStatus   int
		expectResponse []byte
	}

	testcases := []testconfig{{
		name:         "when querying for an existing domain",
		query:        exampleComQuery,
		expectStatus: 200,
		expectResponse: []byte{
			0x00, 0x01, 0x80, 0x00, 0x00, 0x01, 0x00, 0x02,
			0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x78, 0x61,
			0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d,
			0x00, 0x00, 0x01, 0x00, 0x01, 0x07, 0x65, 0x78,
			0x61, 0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f,
			0x6d, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00,
			0x0e, 0x10, 0x00, 0x04, 0x5d, 0xb8, 0xd8, 0x22,
			0x07, 0x65, 0x78, 0x61, 0x6d, 0x70, 0x6c, 0x65,
			0x03, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x05, 0x00,
			0x01, 0x00, 0x00, 0x0e, 0x10, 0x00, 0x13, 0x05,
			0x77, 0x65, 0x62, 0x30, 0x31, 0x07, 0x65, 0x78,
			0x61, 0x6d, 0x70, 0x6c, 0x65, 0x03, 0x63, 0x6f,
			0x6d, 0x00,
		},
	}, {
		name:         "when querying for a nonexisting domain",
		query:        exampleOrgQuery,
		expectStatus: 200,
		expectResponse: []byte{
			0x00, 0x01, 0x80, 0x03, 0x00, 0x01, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x07, 0x65, 0x78, 0x61,
			0x6d, 0x70, 0x6c, 0x65, 0x03, 0x6f, 0x72, 0x67,
			0x00, 0x00, 0x01, 0x00, 0x01,
		},
	}, {
		name:           "with invalid query",
		query:          []byte{0x22},
		expectStatus:   500,
		expectResponse: []byte{},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
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
