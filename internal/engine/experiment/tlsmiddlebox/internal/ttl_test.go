package internal

import (
	"context"
	"net"
	"strings"
	"testing"
)

// add more tests for ttl

// replace this with a stronger test
func TestSetTTL(t *testing.T) {
	var d net.Dialer
	ctx := context.Background()
	conn, err := d.DialContext(ctx, "tcp", "1.1.1.1:80")
	if err != nil {
		t.Fatal("invalid conn")
	}
	// test TTL set
	err = SetConnTTL(conn, 1)
	if err != nil {
		t.Fatal("unexpected error in setting TTL", err)
	}
	var buf [512]byte
	_, err = conn.Write([]byte("1111"))
	if err != nil {
		t.Fatal("error writing", err)
	}
	r, _ := conn.Read(buf[:])
	if r != 0 {
		t.Fatal("unexpected output of size", r)
	}
	// test TTL reset
	ResetConnTTL(conn)
	conn.Close()
	_, err = conn.Read(buf[:])
	expectedFailure := "use of closed network connection"
	if err == nil || !strings.Contains(err.Error(), expectedFailure) {
		t.Fatal("failed to reset TTL")
	}
}
