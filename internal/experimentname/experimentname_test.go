// Package experimentname contains code to manipulate experiment names.
package experimentname

import "testing"

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{
			input:  "example",
			expect: "example",
		},
		{
			input:  "Example",
			expect: "example",
		},
		{
			input:  "ndt7",
			expect: "ndt",
		},
		{
			input:  "Ndt7",
			expect: "ndt",
		},
		{
			input:  "DNSCheck",
			expect: "dnscheck",
		},
		{
			input:  "dns_check",
			expect: "dnscheck",
		},
		{
			input:  "STUNReachability",
			expect: "stunreachability",
		},
		{
			input:  "stun_reachability",
			expect: "stunreachability",
		},
		{
			input:  "WebConnectivity@v0.5",
			expect: "web_connectivity@v0.5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := Canonicalize(tt.input); got != tt.expect {
				t.Errorf("Canonicalize() = %v, want %v", got, tt.expect)
			}
		})
	}
}
