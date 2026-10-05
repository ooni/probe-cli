package netxlite

import (
	"net"
	"testing"
)

func TestGetSystemResolverAddress(t *testing.T) {
	// This reads the real /etc/resolv.conf, so rather than asserting a specific
	// address we assert the invariants that must hold in either environment.
	addr, ok := getSystemResolverAddress()
	if ok {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatal("returned address is not host:port:", addr, err)
		}
		if host == "" || port == "" {
			t.Fatal("returned address has empty host or port:", addr)
		}
	} else if addr != "" {
		t.Fatal("expected empty address when ok is false, got:", addr)
	}
}

func TestDefaultTProxyGetSystemResolverAddress(t *testing.T) {
	// The default tproxy must delegate to getSystemResolverAddress.
	expectedAddr, expectedOK := getSystemResolverAddress()
	addr, ok := (&DefaultTProxy{}).GetSystemResolverAddress()
	if ok != expectedOK || addr != expectedAddr {
		t.Fatal("DefaultTProxy did not delegate to getSystemResolverAddress")
	}
}
