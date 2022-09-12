package tlsmiddlebox

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPrepareAddrs(t *testing.T) {
	type arg struct {
		addrs []string
		port  string
	}
	tests := []struct {
		name string
		args arg
		want []string
	}{{
		name: "with valid input",
		args: arg{
			addrs: []string{"1.1.1.1", "2001:4860:4860::8844"},
			port:  "",
		},
		want: []string{"1.1.1.1:443", "[2001:4860:4860::8844]:443"},
	}, {
		name: "with invalid input",
		args: arg{
			addrs: []string{"1.1.1.1.1", "2001:4860:4860::8844"},
			port:  "",
		},
		want: []string{"[2001:4860:4860::8844]:443"},
	}, {
		name: "with custom port",
		args: arg{
			addrs: []string{"1.1.1.1", "2001:4860:4860::8844"},
			port:  "80",
		},
		want: []string{"1.1.1.1:80", "[2001:4860:4860::8844]:80"},
	}}
	for _, tt := range tests {
		out := prepareAddrs(tt.args.addrs, tt.args.port)
		if diff := cmp.Diff(out, tt.want); diff != "" {
			t.Fatal(diff)
		}
	}
}
