package archival

import "testing"

func TestDNSQueryIPOfType(t *testing.T) {
	type expectation struct {
		qtype  dnsQueryType
		ip     string
		output bool
	}
	var expectations = []expectation{{
		qtype:  "A",
		ip:     "8.8.8.8",
		output: true,
	}, {
		qtype:  "A",
		ip:     "2a00:1450:4002:801::2004",
		output: false,
	}, {
		qtype:  "AAAA",
		ip:     "8.8.8.8",
		output: false,
	}, {
		qtype:  "AAAA",
		ip:     "2a00:1450:4002:801::2004",
		output: true,
	}, {
		qtype:  "ANTANI",
		ip:     "2a00:1450:4002:801::2004",
		output: false,
	}, {
		qtype:  "ANTANI",
		ip:     "8.8.8.8",
		output: false,
	}}
	for _, exp := range expectations {
		if exp.qtype.ipoftype(exp.ip) != exp.output {
			t.Fatalf("failure for %+v", exp)
		}
	}
}
