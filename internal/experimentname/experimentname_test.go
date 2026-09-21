// Package experimentname contains code to manipulate experiment names.
package experimentname

import "testing"

func TestParseString(t *testing.T) {
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
			if got := Parse(tt.input).String(); got != tt.expect {
				t.Errorf("Parse().String() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input      string
		expectBase string
		expectVer  string
		expectStr  string
	}{
		{"example", "example", "", "example"},
		{"Example", "example", "", "example"},
		{"ndt7", "ndt", "", "ndt"},
		{"DNSCheck", "dnscheck", "", "dnscheck"},
		// The version selector must survive verbatim: canonicalizing the whole
		// string would mangle "v0.5" into "v_0_5".
		{"web_connectivity@v0.5", "web_connectivity", "v0.5", "web_connectivity@v0.5"},
		{"WebConnectivity@v0.5", "web_connectivity", "v0.5", "web_connectivity@v0.5"},
		{"facebook_messenger@v0.5", "facebook_messenger", "v0.5", "facebook_messenger@v0.5"},
		// An empty version after "@" collapses to the unversioned form.
		{"example@", "example", "", "example"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Parse(tt.input)
			if got.Base != tt.expectBase {
				t.Errorf("Parse(%q).Base = %q, want %q", tt.input, got.Base, tt.expectBase)
			}
			if got.Version != tt.expectVer {
				t.Errorf("Parse(%q).Version = %q, want %q", tt.input, got.Version, tt.expectVer)
			}
			if got.String() != tt.expectStr {
				t.Errorf("Parse(%q).String() = %q, want %q", tt.input, got.String(), tt.expectStr)
			}
		})
	}
}
