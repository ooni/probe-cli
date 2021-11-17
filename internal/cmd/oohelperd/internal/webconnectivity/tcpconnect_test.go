package webconnectivity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func Test_tcpMapFailure(t *testing.T) {
	tests := []struct {
		name    string
		failure *string
		want    *string
	}{{
		name:    "nil",
		failure: nil,
		want:    nil,
	}, {
		name:    "timeout",
		failure: stringPointerForString(netxlite.FailureGenericTimeoutError),
		want:    stringPointerForString(netxlite.FailureGenericTimeoutError),
	}, {
		name:    "connection refused",
		failure: stringPointerForString(netxlite.FailureConnectionRefused),
		want:    stringPointerForString("connection_refused_error"),
	}, {
		name:    "anything else",
		failure: stringPointerForString(netxlite.FailureEOFError),
		want:    stringPointerForString("connect_error"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tcpMapFailure(tt.failure)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
