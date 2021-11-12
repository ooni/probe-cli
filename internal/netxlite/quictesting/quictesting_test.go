package quictesting

import (
	"net"
	"testing"
)

func TestWorksAsIntended(t *testing.T) {
	epnt := Endpoint("443")
	addr, port, err := net.SplitHostPort(epnt)
	if err != nil {
		t.Fatal(err)
	}
	if addr != Address {
		t.Fatal("invalid addr")
	}
	if port != "443" {
		t.Fatal("invalid port")
	}
}
