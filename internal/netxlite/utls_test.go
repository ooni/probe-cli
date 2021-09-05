package netxlite

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

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
