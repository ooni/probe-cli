package tlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// stopAndWait waits for all backgroung goroutines to join. We only
// use this method in the tests to be sure we don't leak any goroutine.
func (svc *Service) stopAndWait() {
	svc.Stop()
	svc.wg.Wait()
}

func dial() (net.Conn, error) {
	return net.Dial("tcp", "8.8.8.8:853")
}

func TestSuccessfulHandshake(t *testing.T) {
	tcpConn, err := dial()
	if err != nil {
		t.Fatal(err)
	}
	th := New()
	th.StartN(10)
	defer th.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	conn, err := th.Handshake(ctx, &HandshakeRequest{
		Conn: tcpConn,
		Config: &tls.Config{
			NextProtos: []string{"dot"},
			RootCAs:    tlsx.NewDefaultCertPool(),
			ServerName: "dns.google",
		},
		Logger: log.Log,
		Saver:  saver,
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("nil conn?!")
	}
	defer conn.Close()
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}

func TestFailingDial(t *testing.T) {
	tcpConn, err := dial()
	if err != nil {
		t.Fatal(err)
	}
	th := New()
	th.StartN(10)
	defer th.stopAndWait()
	saver := &trace.Saver{}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Microsecond)
	defer cancel()
	conn, err := th.Handshake(ctx, &HandshakeRequest{
		Conn: tcpConn,
		Config: &tls.Config{
			NextProtos: []string{"dot"},
			RootCAs:    tlsx.NewDefaultCertPool(),
			ServerName: "example.com",
		},
		Logger: log.Log,
		Saver:  saver,
	})
	if err == nil || err.Error() != "ssl_invalid_hostname" {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("returned a valid conn?!")
	}
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}
