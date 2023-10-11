package testingquic

import (
	"net"
	"testing"
)

func TestWorksAsIntended(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	endpoint := MustEndpoint("443")
	addr, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		t.Fatal(err)
	}

	if addr != address {
		t.Fatal("invalid addr")
	}
	if port != "443" {
		t.Fatal("invalid port")
	}
}
