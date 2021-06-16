package httptransport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"testing"

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
	return net.Dial("tcp", "www.google.com:80")
}

func TestSuccessfulRoundTripHTTP(t *testing.T) {
	conn, err := dial()
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://www.google.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	txp := New()
	txp.StartN(3)
	defer txp.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	resp, err := txp.RoundTrip(ctx, &RoundTripRequest{
		Req:    req,
		Conn:   conn,
		Logger: log.Log,
		Saver:  saver,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("nil resp?!")
	}
	defer resp.Body.Close()
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}

func TestRoundTripWithClosedConn(t *testing.T) {
	conn, err := dial()
	if err != nil {
		t.Fatal(err)
	}
	conn.Close() // immediately close the connection
	req, err := http.NewRequest("GET", "http://www.google.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	txp := New()
	txp.StartN(3)
	defer txp.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	resp, err := txp.RoundTrip(ctx, &RoundTripRequest{
		Req:    req,
		Conn:   conn,
		Logger: log.Log,
		Saver:  saver,
	})
	if err == nil || !strings.HasSuffix(err.Error(), "use of closed network connection") {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("non-nil resp?!")
	}
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}

func dialTLS() (net.Conn, error) {
	return tls.Dial("tcp", "8.8.8.8:443", &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    tlsx.NewDefaultCertPool(),
		ServerName: "dns.google",
	})
}

func TestSuccessfulRoundTripHTTPS(t *testing.T) {
	conn, err := dialTLS()
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "https://dns.google/", nil)
	if err != nil {
		t.Fatal(err)
	}
	txp := New()
	txp.StartN(3)
	defer txp.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	resp, err := txp.RoundTrip(ctx, &RoundTripRequest{
		Req:    req,
		Conn:   conn,
		Logger: log.Log,
		Saver:  saver,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("nil resp?!")
	}
	defer resp.Body.Close()
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}
