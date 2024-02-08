package webconnectivityalgo

import "testing"

func TestRandomDNSOverUDPResolverEndpointIPv4(t *testing.T) {
	results := make(map[string]int64)
	const maxruns = 1024
	for idx := 0; idx < maxruns; idx++ {
		endpoint := RandomDNSOverUDPResolverEndpointIPv4()
		results[endpoint]++
	}
	t.Log(results)
	if len(results) < 3 {
		t.Fatal("expected to see at least three different results out of 1024 runs")
	}
}
