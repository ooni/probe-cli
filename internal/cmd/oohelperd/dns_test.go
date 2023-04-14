package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// stringPointerForString is an helper function to map a string
// to a pointer to the same string.
func stringPointerForString(s string) *string {
	return &s
}

// Test_dnsMapFailure ensures that we are mapping OONI failure
// strings to the strings the legacy TH would have returned.
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

// TestDNSDo contains unit tests for [dnsDo].
func TestDNSDo(t *testing.T) {

	type testcase struct {
		// name is the name of the test case
		name string

		// inputDomain is the domain to resolve
		inputDomain string

		// inputNewResolver is the factory to create a new resolver.
		inputNewResolver func(model.Logger) model.Resolver

		// expectFailure is the expected failure
		expectFailure *string

		// expectAddrs contains the addrs we expecy to see
		expectAddrs []string
	}

	var testcases = []testcase{{
		name:        "returns non-nil, empty addresses list on NXDOMAIN",
		inputDomain: "www.ooni.nonexistent",
		inputNewResolver: func(model.Logger) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, hostname string) ([]string, error) {
					return nil, errors.New(netxlite.DNSNoSuchHostSuffix)
				},
				MockCloseIdleConnections: func() {
					// nothing
				},
			}
		},
		expectFailure: stringPointerForString(model.THDNSNameError),
		expectAddrs:   []string{},
	}, {
		name:        "returns the expected result in case of successful lookup",
		inputDomain: "www.ooni.org",
		inputNewResolver: func(model.Logger) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8", "8.8.4.4"}, nil
				},
				MockCloseIdleConnections: func() {
					// nothing
				},
			}
		},
		expectFailure: nil,
		expectAddrs:   []string{"8.8.8.8", "8.8.4.4"},
	}, {
		name:        "when there is no answer",
		inputDomain: "www.ooni.org",
		inputNewResolver: func(model.Logger) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, errors.New(netxlite.DNSNoAnswerSuffix)
				},
				MockCloseIdleConnections: func() {
					// nothing
				},
			}
		},
		expectFailure: nil,
		expectAddrs:   []string{},
	}, {
		name:        "when the server is misbehaving",
		inputDomain: "www.ooni.org",
		inputNewResolver: func(model.Logger) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, errors.New(netxlite.DNSServerMisbehavingSuffix)
				},
				MockCloseIdleConnections: func() {
					// nothing
				},
			}
		},
		expectFailure: stringPointerForString("dns_server_failure"),
		expectAddrs:   []string{},
	}, {
		name:        "for any other error",
		inputDomain: "www.ooni.org",
		inputNewResolver: func(model.Logger) model.Resolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, io.EOF
				},
				MockCloseIdleConnections: func() {
					// nothing
				},
			}
		},
		expectFailure: stringPointerForString("unknown_error"),
		expectAddrs:   []string{},
	}}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// prepare configuration for testing
			config := &dnsConfig{
				Domain:      tt.inputDomain,
				Logger:      model.DiscardLogger,
				NewResolver: tt.inputNewResolver,
				Out:         make(chan model.THDNSResult, 1),
				Wg:          &sync.WaitGroup{},
			}

			// run the micro-measurement in the background, wait for it
			// to complete, and obtain the results.
			config.Wg.Add(1)
			dnsDo(ctx, config)
			config.Wg.Wait()
			resp := <-config.Out

			// compare the results with the expectations.
			if diff := cmp.Diff(tt.expectFailure, resp.Failure); diff != "" {
				t.Fatal(diff)
			}
			if diff := cmp.Diff(tt.expectAddrs, resp.Addrs); diff != "" {
				t.Fatal(diff)
			}
			expectASNs := []int64{} // should be unused!
			if diff := cmp.Diff(expectASNs, resp.ASNs); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
