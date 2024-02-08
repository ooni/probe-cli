package webconnectivityalgo

import (
	"net"
	"testing"
)

func TestRandomDNSOverUDPResolverEndpointIPv4(t *testing.T) {
	results := make(map[string]int64)
	const maxruns = 1024
	for idx := 0; idx < maxruns; idx++ {
		endpoint := RandomDNSOverUDPResolverEndpointIPv4()
		results[endpoint]++
		if _, _, err := net.SplitHostPort(endpoint); err != nil {
			t.Fatal(err)
		}
	}
	t.Log(results)
	if len(results) < 3 {
		t.Fatal("expected to see at least three different results out of 1024 runs")
	}
}
