package tlsmiddlebox

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ethernetMSS is the TCP payload of a full segment on a 1500-byte MTU path with TCP
// timestamps (1500 - 20 IP - 20 TCP - 12 options), as in the netns tests' captures.
const ethernetMSS = 1448

// clientHelloRecord returns what the experiment writes for its ClientHello with the given
// client ID (0: Go's crypto/tls), as handshakeWithTTL builds it: the whole TLS record, header
// included. It handshakes over a conn that records writes and fails the first read.
func clientHelloRecord(t *testing.T, clientID int, sni string) []byte {
	t.Helper()
	var written []byte
	addr := &net.TCPAddr{IP: net.IPv4(10, 0, 4, 2), Port: 443}
	conn := &mocks.Conn{
		MockWrite: func(b []byte) (int, error) {
			written = append(written, b...)
			return len(b), nil
		},
		MockRead: func(b []byte) (int, error) {
			return 0, errors.New("no server: stop after the ClientHello")
		},
		MockClose:            func() error { return nil },
		MockLocalAddr:        func() net.Addr { return &net.TCPAddr{IP: net.IPv4(10, 0, 1, 2), Port: 40000} },
		MockRemoteAddr:       func() net.Addr { return addr },
		MockSetDeadline:      func(time.Time) error { return nil },
		MockSetReadDeadline:  func(time.Time) error { return nil },
		MockSetWriteDeadline: func(time.Time) error { return nil },
	}
	trace := measurexlite.NewTrace(0, time.Now())
	thx := trace.NewTLSHandshakerStdlib(model.DiscardLogger)
	if clientID > 0 {
		thx = trace.NewTLSHandshakerUTLS(model.DiscardLogger, ClientIDs[clientID])
	}
	if _, err := thx.Handshake(context.Background(), conn, genTLSConfig(sni)); err == nil {
		t.Fatal("the handshake succeeded without a server")
	}
	if len(written) < 5 || written[0] != 0x16 {
		t.Fatalf("client %d: wrote %d bytes, not a TLS handshake record", clientID, len(written))
	}
	return written
}

// TestClientHelloSegments pins how many TCP segments each ClientHello takes, and in which the
// SNI is, because that decides the TTL at which a middlebox that matches the SNI shows up in a
// trace (TestNetns): with one segment, at the middlebox's hop; with two, one hop later, as the
// segment without the SNI then expires at the next router. A uTLS or Go update can change it:
// since #1862 (refraction-networking/utls 1.8.2) uTLS Chrome carries the X25519MLKEM768 key
// share, 1.7-1.8 kB, and shuffles its extensions, so its SNI is in the first segment of some
// hellos and in the second of others (210 of 500 sampled, 2026-11-26).
func TestClientHelloSegments(t *testing.T) {
	cases := []struct {
		name     string
		clientID int
		segments int
	}{
		{"Go's crypto/tls", 0, 2}, // X25519MLKEM768 by default
		{"uTLS Go", 1, 2},
		{"uTLS Chrome", 2, 2},
		{"uTLS Firefox", 3, 1},
		{"uTLS iOS", 4, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Chrome's hellos vary: look at 50 of them, the others are fixed
			n := 1
			if tc.clientID == 2 {
				n = 50
			}
			first, second := 0, 0
			for i := 0; i < n; i++ {
				rec := clientHelloRecord(t, tc.clientID, "target.test")
				if got := (len(rec) + ethernetMSS - 1) / ethernetMSS; got != tc.segments {
					t.Fatalf("%d bytes: %d segment(s), want %d", len(rec), got, tc.segments)
				}
				if bytes.Contains(rec[:min(len(rec), ethernetMSS)], []byte("target.test")) {
					first++
				} else {
					second++
				}
			}
			switch {
			case tc.clientID == 2 && (first == 0 || second == 0):
				// the chance of 50 hellos all alike while uTLS shuffles is about 1 in 10^11
				t.Errorf("SNI in the first segment of %d hellos, in the second of %d: "+
					"uTLS Chrome no longer shuffles its extensions?", first, second)
			case tc.clientID != 2 && second > 0:
				t.Errorf("the SNI is not in the first segment")
			}
		})
	}
}
