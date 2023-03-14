package dnsreport

import "testing"

func TestHTTPTransportNetwork(t *testing.T) {
	if value := defaultHTTPTransport.Network(); value != "tcp" {
		t.Fatal("expected tcp, got", value)
	}
}
