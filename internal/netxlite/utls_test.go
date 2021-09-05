package netxlite

import (
	"context"
	"crypto/tls"
	"net"
	"testing"

	utls "gitlab.com/yawning/utls.git"
)

func TestUTLSHandshakerChrome(t *testing.T) {
	h := &TLSHandshakerConfigurable{
		NewConn: NewConnUTLS(&utls.HelloChrome_Auto),
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
