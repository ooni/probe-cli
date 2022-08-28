package main

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func stringPointerForString(s string) *string {
	return &s
}

func Test_dnsMapFailure(t *testing.T) {
	tests := []struct {
		name    string
		failure *string
		want    *string
	}{{
		name:    "nil",
		failure: nil,
		want:    nil,
	}, {
		name:    "nxdomain",
		failure: stringPointerForString(netxlite.FailureDNSNXDOMAINError),
		want:    stringPointerForString(model.THDNSNameError),
	}, {
		name:    "no answer",
		failure: stringPointerForString(netxlite.FailureDNSNoAnswer),
		want:    nil,
	}, {
		name:    "non recoverable failure",
		failure: stringPointerForString(netxlite.FailureDNSNonRecoverableFailure),
		want:    stringPointerForString("dns_server_failure"),
	}, {
		name:    "refused",
		failure: stringPointerForString(netxlite.FailureDNSRefusedError),
		want:    stringPointerForString("dns_server_failure"),
	}, {
		name:    "server misbehaving",
		failure: stringPointerForString(netxlite.FailureDNSServerMisbehaving),
		want:    stringPointerForString("dns_server_failure"),
	}, {
		name:    "temporary failure",
		failure: stringPointerForString(netxlite.FailureDNSTemporaryFailure),
		want:    stringPointerForString("dns_server_failure"),
	}, {
		name:    "anything else",
		failure: stringPointerForString(netxlite.FailureEOFError),
		want:    stringPointerForString("unknown_error"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dnsMapFailure(tt.failure)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestDNSDo(t *testing.T) {
	t.Run("returns non-nil addresses list on nxdomin", func(t *testing.T) {
		ctx := context.Background()
		config := &dnsConfig{
			Domain: "antani.ooni.org",
			NewResolver: func() model.Resolver {
				return &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, netxlite.ErrOODNSNoSuchHost
					},
					MockCloseIdleConnections: func() {
						// nothing
					},
				}
			},
			Out: make(chan model.THDNSResult, 1),
			Wg:  &sync.WaitGroup{},
		}
		config.Wg.Add(1)
		dnsDo(ctx, config)
		config.Wg.Wait()
		resp := <-config.Out
		if resp.Addrs == nil || len(resp.Addrs) != 0 {
			t.Fatal("returned nil Addrs or Addrs containing replies")
		}
	})
}
