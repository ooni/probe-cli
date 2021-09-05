package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	utls "gitlab.com/yawning/utls.git"
)

func TestUTLSHandshakerChrome(t *testing.T) {
	h := &tlsHandshakerConfigurable{
		NewConn: newConnUTLS(&utls.HelloChrome_Auto),
	}
	cfg := &tls.Config{ServerName: "google.com"}
	conn, err := net.Dial("tcp", "google.com:443")
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	conn, _, err = h.Handshake(context.Background(), conn, cfg)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if conn == nil {
		t.Fatal("nil connection")
	}
}

func TestNewTLSHandshakerUTLSTypes(t *testing.T) {
	th := NewTLSHandshakerUTLS(log.Log, &utls.HelloChrome_83)
	thl, okay := th.(*tlsHandshakerLogger)
	if !okay {
		t.Fatal("invalid type")
	}
	if thl.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	thc, okay := thl.TLSHandshaker.(*tlsHandshakerConfigurable)
	if !okay {
		t.Fatal("invalid type")
	}
	if thc.NewConn == nil {
		t.Fatal("expected non-nil NewConn")
	}
}

func TestUTLSConnHandshakeNotInterruptedSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &utlsConn{
		testableHandshake: func() error {
			return nil
		},
	}
	err := conn.HandshakeContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUTLSConnHandshakeNotInterruptedFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	conn := &utlsConn{
		testableHandshake: func() error {
			return expected
		},
	}
	err := conn.HandshakeContext(ctx)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
}

func TestUTLSConnHandshakeInterrupted(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	sigch := make(chan interface{})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	conn := &utlsConn{
		testableHandshake: func() error {
			defer wg.Done()
			<-sigch
			return nil
		},
	}
	err := conn.HandshakeContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected", err)
	}
	close(sigch)
	wg.Wait()
}

func TestUTLSConnHandshakePanic(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx := context.Background()
	conn := &utlsConn{
		testableHandshake: func() error {
			defer wg.Done()
			panic("mascetti")
		},
	}
	err := conn.HandshakeContext(ctx)
	if !errors.Is(err, ErrUTLSHandshakePanic) {
		t.Fatal("not the error we expected", err)
	}
	wg.Wait()
}
