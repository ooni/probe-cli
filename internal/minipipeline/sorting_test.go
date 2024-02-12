package minipipeline

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSortDNSLookupResults(t *testing.T) {
	newfailurestring := func(s string) *string {
		return &s
	}

	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalDNSLookupResult
		expect   []*model.ArchivalDNSLookupResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return nil
		},
		expect: []*model.ArchivalDNSLookupResult{},
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return []*model.ArchivalDNSLookupResult{}
		},
		expect: []*model.ArchivalDNSLookupResult{},
	}, {
		name: "with good input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return []*model.ArchivalDNSLookupResult{
				{
					Engine:          "udp",
					Failure:         newfailurestring("dns_no_answer"),
					QueryType:       "AAAA",
					ResolverAddress: "1.1.1.1:53",
					TransactionID:   5,
				},
				{
					Engine:          "udp",
					Failure:         nil,
					QueryType:       "A",
					ResolverAddress: "1.1.1.1:53",
					TransactionID:   5,
				},
				{
					Engine:          "udp",
					Failure:         newfailurestring("dns_no_answer"),
					QueryType:       "AAAA",
					ResolverAddress: "8.8.8.8:53",
					TransactionID:   3,
				},
				{
					Engine:          "udp",
					Failure:         nil,
					QueryType:       "A",
					ResolverAddress: "8.8.8.8:53",
					TransactionID:   3,
				},
				{
					Engine:          "doh",
					Failure:         newfailurestring("dns_no_answer"),
					QueryType:       "AAAA",
					ResolverAddress: "https://dns.google/dns-query",
					TransactionID:   2,
				},
				{
					Engine:          "doh",
					Failure:         nil,
					QueryType:       "A",
					ResolverAddress: "https://dns.google/dns-query",
					TransactionID:   2,
				},
				{
					Engine:        "getaddrinfo",
					QueryType:     "ANY",
					Failure:       nil,
					TransactionID: 1,
				},
				{
					Engine:        "getaddrinfo",
					QueryType:     "ANY",
					Failure:       nil,
					TransactionID: 7,
				},
			}
		},
		expect: []*model.ArchivalDNSLookupResult{
			{
				Engine:          "doh",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "https://dns.google/dns-query",
				TransactionID:   2,
			},
			{
				Engine:          "doh",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "https://dns.google/dns-query",
				TransactionID:   2,
			},
			{
				Engine:        "getaddrinfo",
				QueryType:     "ANY",
				Failure:       nil,
				TransactionID: 1,
			},
			{
				Engine:        "getaddrinfo",
				QueryType:     "ANY",
				Failure:       nil,
				TransactionID: 7,
			},
			{
				Engine:          "udp",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "8.8.8.8:53",
				TransactionID:   3,
			},
			{
				Engine:          "udp",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "8.8.8.8:53",
				TransactionID:   3,
			},
			{
				Engine:          "udp",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "1.1.1.1:53",
				TransactionID:   5,
			},
			{
				Engine:          "udp",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "1.1.1.1:53",
				TransactionID:   5,
			},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.inputGen()
			output := SortDNSLookupResults(input)

			t.Run("the input should not have mutated", func(t *testing.T) {
				if diff := cmp.Diff(tc.inputGen(), input); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("the output should be consistent with expectations", func(t *testing.T) {
				if diff := cmp.Diff(tc.expect, output); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}

func TestSortNetworkEvents(t *testing.T) {
	newfailurestring := func(s string) *string {
		return &s
	}

	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalNetworkEvent
		expect   []*model.ArchivalNetworkEvent
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return nil
		},
		expect: []*model.ArchivalNetworkEvent{},
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return []*model.ArchivalNetworkEvent{}
		},
		expect: []*model.ArchivalNetworkEvent{},
	}, {
		name: "with good input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return []*model.ArchivalNetworkEvent{
				{
					Address:       "8.8.8.8:443",
					Failure:       newfailurestring("connection_reset"),
					Operation:     "read",
					T:             1.1,
					TransactionID: 5,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					Operation:     "write",
					T:             1.0,
					TransactionID: 5,
				},
				{
					Address:       "1.1.1.1:443",
					Failure:       newfailurestring("eof_error"),
					Operation:     "read",
					T:             0.9,
					TransactionID: 3,
				},
				{
					Address:       "1.1.1.1:443",
					Failure:       nil,
					Operation:     "write",
					T:             0.4,
					TransactionID: 3,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					Operation:     "write",
					T:             1.4,
					TransactionID: 2,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					Operation:     "read",
					T:             1.5,
					TransactionID: 2,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					Operation:     "write",
					T:             1.4,
					TransactionID: 3,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					Operation:     "read",
					T:             1.5,
					TransactionID: 3,
				},
			}
		},
		expect: []*model.ArchivalNetworkEvent{
			{
				Address:       "1.1.1.1:443",
				Failure:       nil,
				Operation:     "write",
				T:             0.4,
				TransactionID: 3,
			},
			{
				Address:       "1.1.1.1:443",
				Failure:       newfailurestring("eof_error"),
				Operation:     "read",
				T:             0.9,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				Operation:     "write",
				T:             1.4,
				TransactionID: 2,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				Operation:     "read",
				T:             1.5,
				TransactionID: 2,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				Operation:     "write",
				T:             1.4,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				Operation:     "read",
				T:             1.5,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				Operation:     "write",
				T:             1.0,
				TransactionID: 5,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       newfailurestring("connection_reset"),
				Operation:     "read",
				T:             1.1,
				TransactionID: 5,
			},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.inputGen()
			output := SortNetworkEvents(input)

			t.Run("the input should not have mutated", func(t *testing.T) {
				if diff := cmp.Diff(tc.inputGen(), input); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("the output should be consistent with expectations", func(t *testing.T) {
				if diff := cmp.Diff(tc.expect, output); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}

func TestSortTCPConnectResults(t *testing.T) {
	newfailurestring := func(s string) *string {
		return &s
	}

	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalTCPConnectResult
		expect   []*model.ArchivalTCPConnectResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return nil
		},
		expect: []*model.ArchivalTCPConnectResult{},
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return []*model.ArchivalTCPConnectResult{}
		},
		expect: []*model.ArchivalTCPConnectResult{},
	}, {
		name: "with good input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return []*model.ArchivalTCPConnectResult{
				{
					IP:   "1.1.1.1",
					Port: 443,
					Status: model.ArchivalTCPConnectStatus{
						Failure: newfailurestring("connection_reset"),
					},
					T:             0.9,
					TransactionID: 7,
				},
				{
					IP:   "8.8.8.8",
					Port: 443,
					Status: model.ArchivalTCPConnectStatus{
						Failure: newfailurestring("connection_reset"),
					},
					T:             1.1,
					TransactionID: 5,
				},
				{
					IP:   "8.8.8.8",
					Port: 80,
					Status: model.ArchivalTCPConnectStatus{
						Failure: newfailurestring("connection_reset"),
					},
					T:             1.1,
					TransactionID: 5,
				},
				{
					IP:   "1.1.1.1",
					Port: 443,
					Status: model.ArchivalTCPConnectStatus{
						Failure: newfailurestring("connection_reset"),
					},
					T:             0.9,
					TransactionID: 3,
				},
				{
					IP:   "8.8.8.8",
					Port: 443,
					Status: model.ArchivalTCPConnectStatus{
						Failure: nil,
					},
					T:             1.4,
					TransactionID: 2,
				},
				{
					IP:   "8.8.8.8",
					Port: 443,
					Status: model.ArchivalTCPConnectStatus{
						Failure: nil,
					},
					T:             1.4,
					TransactionID: 3,
				},
				{
					IP:   "8.8.8.8",
					Port: 80,
					Status: model.ArchivalTCPConnectStatus{
						Failure: newfailurestring("connection_reset"),
					},
					T:             5.1,
					TransactionID: 5,
				},
			}
		},
		expect: []*model.ArchivalTCPConnectResult{
			{
				IP:   "1.1.1.1",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Failure: newfailurestring("connection_reset"),
				},
				T:             0.9,
				TransactionID: 3,
			},
			{
				IP:   "1.1.1.1",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Failure: newfailurestring("connection_reset"),
				},
				T:             0.9,
				TransactionID: 7,
			},
			{
				IP:   "8.8.8.8",
				Port: 80,
				Status: model.ArchivalTCPConnectStatus{
					Failure: newfailurestring("connection_reset"),
				},
				T:             1.1,
				TransactionID: 5,
			},
			{
				IP:   "8.8.8.8",
				Port: 80,
				Status: model.ArchivalTCPConnectStatus{
					Failure: newfailurestring("connection_reset"),
				},
				T:             5.1,
				TransactionID: 5,
			},
			{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Failure: nil,
				},
				T:             1.4,
				TransactionID: 2,
			},
			{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Failure: nil,
				},
				T:             1.4,
				TransactionID: 3,
			},
			{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Failure: newfailurestring("connection_reset"),
				},
				T:             1.1,
				TransactionID: 5,
			},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.inputGen()
			output := SortTCPConnectResults(input)

			t.Run("the input should not have mutated", func(t *testing.T) {
				if diff := cmp.Diff(tc.inputGen(), input); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("the output should be consistent with expectations", func(t *testing.T) {
				if diff := cmp.Diff(tc.expect, output); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}

func TestSortQUICTLSHandshakeResults(t *testing.T) {
	newfailurestring := func(s string) *string {
		return &s
	}

	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalTLSOrQUICHandshakeResult
		expect   []*model.ArchivalTLSOrQUICHandshakeResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return nil
		},
		expect: []*model.ArchivalTLSOrQUICHandshakeResult{},
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return []*model.ArchivalTLSOrQUICHandshakeResult{}
		},
		expect: []*model.ArchivalTLSOrQUICHandshakeResult{},
	}, {
		name: "with good input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return []*model.ArchivalTLSOrQUICHandshakeResult{
				{
					Address:       "8.8.8.8:443",
					Failure:       newfailurestring("connection_reset"),
					T:             1.1,
					TransactionID: 5,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					T:             1.0,
					TransactionID: 5,
				},
				{
					Address:       "1.1.1.1:443",
					Failure:       newfailurestring("eof_error"),
					T:             0.9,
					TransactionID: 3,
				},
				{
					Address:       "1.1.1.1:443",
					Failure:       nil,
					T:             0.4,
					TransactionID: 3,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					T:             1.4,
					TransactionID: 2,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					T:             1.5,
					TransactionID: 2,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					T:             1.4,
					TransactionID: 3,
				},
				{
					Address:       "8.8.8.8:443",
					Failure:       nil,
					T:             1.5,
					TransactionID: 3,
				},
			}
		},
		expect: []*model.ArchivalTLSOrQUICHandshakeResult{
			{
				Address:       "1.1.1.1:443",
				Failure:       nil,
				T:             0.4,
				TransactionID: 3,
			},
			{
				Address:       "1.1.1.1:443",
				Failure:       newfailurestring("eof_error"),
				T:             0.9,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				T:             1.4,
				TransactionID: 2,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				T:             1.5,
				TransactionID: 2,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				T:             1.4,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				T:             1.5,
				TransactionID: 3,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				T:             1.0,
				TransactionID: 5,
			},
			{
				Address:       "8.8.8.8:443",
				Failure:       newfailurestring("connection_reset"),
				T:             1.1,
				TransactionID: 5,
			},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.inputGen()
			output := SortTLSHandshakeResults(input)

			t.Run("the input should not have mutated", func(t *testing.T) {
				if diff := cmp.Diff(tc.inputGen(), input); diff != "" {
					t.Fatal(diff)
				}
			})

			t.Run("the output should be consistent with expectations", func(t *testing.T) {
				if diff := cmp.Diff(tc.expect, output); diff != "" {
					t.Fatal(diff)
				}
			})
		})
	}
}
