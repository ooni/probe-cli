package netxlite

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestTLSHandshakerStdlibWithError(t *testing.T) {
	var times []time.Time
	h := &TLSHandshakerStdlib{}
	tcpConn := &netxmocks.Conn{
		MockWrite: func(b []byte) (int, error) {
			return 0, io.EOF
		},
		MockSetDeadline: func(t time.Time) error {
			times = append(times, t)
			return nil
		},
	}
	ctx := context.Background()
	conn, _, err := h.Handshake(ctx, tcpConn, &tls.Config{
		ServerName: "x.org",
	})
	if err != io.EOF {
		t.Fatal("not the error that we expected")
	}
	if conn != nil {
		t.Fatal("expected nil con here")
	}
	if len(times) != 2 {
		t.Fatal("expected two time entries")
	}
	if !times[0].After(time.Now()) {
		t.Fatal("timeout not in the future")
	}
	if !times[1].IsZero() {
		t.Fatal("did not clear timeout on exit")
	}
}

func TestTLSHandshakerStdlibSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(200)
	})
	srvr := httptest.NewTLSServer(handler)
	defer srvr.Close()
	URL, err := url.Parse(srvr.URL)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", URL.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	handshaker := &TLSHandshakerStdlib{}
	ctx := context.Background()
	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		ServerName:         URL.Hostname(),
	}
	tlsConn, connState, err := handshaker.Handshake(ctx, conn, config)
	if err != nil {
		t.Fatal(err)
	}
	defer tlsConn.Close()
	if connState.Version != tls.VersionTLS13 {
		t.Fatal("unexpected TLS version")
	}
}
