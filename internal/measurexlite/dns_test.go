package measurexlite

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewResolver(t *testing.T) {
	t.Run("WrapResolver creates a wrapped resolver with Trace", func(t *testing.T) {
		underlying := &mocks.Resolver{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		resolvert := trace.wrapResolver(underlying).(*resolverTrace)
		if resolvert.r != underlying {
			t.Fatal("invalid parallel resolver")
		}
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("Trace-aware resolver forwards underlying functions", func(t *testing.T) {
		var called bool
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		mockResolver := &mocks.Resolver{
			MockAddress: func() string {
				return "dns.google"
			},
			MockNetwork: func() string {
				return "udp"
			},
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return []string{"1.1.1.1"}, nil
			},
			MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
				return &model.HTTPSSvc{
					IPv4: []string{"1.1.1.1"},
				}, nil
			},
			MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
				return []*net.NS{{
					Host: "1.1.1.1",
				}}, nil
			},
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		resolver := trace.wrapResolver(mockResolver)

		t.Run("Address is correctly forwarded", func(t *testing.T) {
			got := resolver.Address()
			if got != "dns.google" {
				t.Fatal("Address not called")
			}
		})

		t.Run("Network is correctly forwarded", func(t *testing.T) {
			got := resolver.Network()
			if got != "udp" {
				t.Fatal("Network not called")
			}
		})

		t.Run("LookupHost is correctly forwarded", func(t *testing.T) {
			want := []string{"1.1.1.1"}
			ctx := context.Background()
			got, err := resolver.LookupHost(ctx, "example.com")
			if err != nil {
				t.Fatal("expected nil error")
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("LookupHTTPS is correctly forwarded", func(t *testing.T) {
			want := &model.HTTPSSvc{
				IPv4: []string{"1.1.1.1"},
			}
			ctx := context.Background()
			got, err := resolver.LookupHTTPS(ctx, "example.com")
			if err != nil {
				t.Fatal("expected nil error")
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("LookupNS is correctly forwarded", func(t *testing.T) {
			want := []*net.NS{{
				Host: "1.1.1.1",
			}}
			ctx := context.Background()
			got, err := resolver.LookupNS(ctx, "example.com")
			if err != nil {
				t.Fatal("expected nil error")
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("CloseIdleConnections is correctly forwarded", func(t *testing.T) {
			resolver.CloseIdleConnections()
			if !called {
				t.Fatal("CloseIdleConnections not called")
			}
		})
	})

	t.Run("LookupHost saves into trace", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now
		txp := &mocks.DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				response := &mocks.DNSResponse{
					MockDecodeLookupHost: func() ([]string, error) {
						if query.Type() != dns.TypeA {
							return []string{"fe80::a00:20ff:feb9:4c54"}, nil
						}
						return []string{"1.1.1.1"}, nil
					},
					MockDecodeCNAME: func() (string, error) {
						return "dns.google.", nil
					},
					MockRcode: func() int {
						return 0
					},
					MockBytes: func() []byte {
						return []byte{}
					},
				}
				return response, nil
			},
			MockRequiresPadding: func() bool {
				return true
			},
			MockNetwork: func() string {
				return "mocked"
			},
			MockAddress: func() string {
				return "dns.google"
			},
		}
		r := netxlite.NewUnwrappedParallelResolver(txp)
		resolver := trace.wrapResolver(r)
		ctx := context.Background()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 2 {
			t.Fatal("unexpected array output", addrs)
		}
		if addrs[0] != "1.1.1.1" && addrs[1] != "1.1.1.1" {
			t.Fatal("unexpected array output", addrs)
		}
		if addrs[0] != "fe80::a00:20ff:feb9:4c54" && addrs[1] != "fe80::a00:20ff:feb9:4c54" {
			t.Fatal("unexpected array output", addrs)
		}

		t.Run("DNSLookup events", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip()
			if len(events) != 2 {
				t.Fatal("unexpected DNS events length")
			}
			for _, ev := range events {
				if ev.ResolverAddress != "dns.google" {
					t.Fatal("unexpected resolver address")
				}
				if ev.Engine != "mocked" {
					t.Fatal("unexpected engine")
				}
				if len(ev.Answers) != 2 {
					t.Fatal("expected single answer in DNSLookup event")
				}
				if ev.QueryType == "A" && ev.Answers[0].IPv4 != "1.1.1.1" {
					t.Fatal("unexpected A query result")
				}
				if ev.QueryType == "AAAA" && ev.Answers[0].IPv6 != "fe80::a00:20ff:feb9:4c54" {
					t.Fatal("unexpected AAAA query result")
				}
				if ev.Answers[1].AnswerType != "CNAME " && ev.Answers[1].Hostname != "dns.google." {
					t.Fatal("unexpected second answer (expected CNAME)", ev.Answers[1])
				}
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 2 {
				t.Fatal("unexpected network events length")
			}
			foundNames := map[string]int{}
			for _, ev := range events {
				foundNames[ev.Operation]++
			}
			if foundNames["resolve_start"] != 1 {
				t.Fatal("missing resolve_start")
			}
			if foundNames["resolve_done"] != 1 {
				t.Fatal("missing resolve_done")
			}
		})
	})

	t.Run("LookupHost discards events when buffers are full", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.dnsLookup = make(chan *model.ArchivalDNSLookupResult) // no buffer
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent) // ditto
		trace.TimeNowFn = td.Now
		txp := &mocks.DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				response := &mocks.DNSResponse{
					MockDecodeLookupHost: func() ([]string, error) {
						if query.Type() != dns.TypeA {
							return []string{"fe80::a00:20ff:feb9:4c54"}, nil
						}
						return []string{"1.1.1.1"}, nil
					},
					MockDecodeCNAME: func() (string, error) {
						return "dns.google.", nil
					},
					MockRcode: func() int {
						return 0
					},
					MockBytes: func() []byte {
						return []byte{}
					},
				}
				return response, nil
			},
			MockRequiresPadding: func() bool {
				return true
			},
			MockNetwork: func() string {
				return ""
			},
			MockAddress: func() string {
				return "dns.google"
			},
		}
		r := netxlite.NewUnwrappedParallelResolver(txp)
		resolver := trace.wrapResolver(r)
		ctx := context.Background()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 2 {
			t.Fatal("unexpected array output", addrs)
		}
		if addrs[0] != "1.1.1.1" && addrs[1] != "1.1.1.1" {
			t.Fatal("unexpected array output", addrs)
		}
		if addrs[0] != "fe80::a00:20ff:feb9:4c54" && addrs[1] != "fe80::a00:20ff:feb9:4c54" {
			t.Fatal("unexpected array output", addrs)
		}

		t.Run("DNSLookup Events", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip()
			if len(events) != 0 {
				t.Fatal("expected to see no DNSLookup events")
			}
		})

		t.Run("Network events", func(t *testing.T) {
			events := trace.NetworkEvents()
			if len(events) != 0 {
				t.Fatal("unexpected to see no network events")
			}
		})
	})
}

func TestNewWrappedResolvers(t *testing.T) {
	t.Run("NewParallelDNSOverHTTPSResolver works as intended", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		resolver := trace.NewParallelDNSOverHTTPSResolver(model.DiscardLogger, "https://dns.google.com")
		resolvert := resolver.(*resolverTrace)
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
		if resolver.Network() != "doh" {
			t.Fatal("unexpected resolver network")
		}
	})

	t.Run("NewParallelUDPResolver works as intended", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		dialer := netxlite.NewDialerWithStdlibResolver(model.DiscardLogger)
		resolver := trace.NewParallelUDPResolver(model.DiscardLogger, dialer, "1.1.1.1:53")
		resolvert := resolver.(*resolverTrace)
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
		if resolver.Network() != "udp" {
			t.Fatal("unexpected resolver network")
		}
	})

	t.Run("NewStdlibResolver works as intended", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		resolver := trace.NewStdlibResolver(model.DiscardLogger)
		resolvert := resolver.(*resolverTrace)
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
		switch network := resolver.Network(); network {
		case netxlite.StdlibResolverGetaddrinfo,
			netxlite.StdlibResolverGolangNetResolver:
		// ok
		default:
			t.Fatal("unexpected resolver network", network)
		}
	})
}

func TestFirstDNSLookup(t *testing.T) {
	t.Run("returns nil when buffer is empty", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		got := trace.FirstDNSLookup()
		if got != nil {
			t.Fatal("expected nil event")
		}
	})

	t.Run("return first non-nil DNSLookup", func(t *testing.T) {
		filler := func(tx *Trace, events []*model.ArchivalDNSLookupResult) {
			for _, ev := range events {
				tx.dnsLookup <- ev
			}
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		expect := []*model.ArchivalDNSLookupResult{{
			Engine:    "doh",
			Failure:   nil,
			Hostname:  "example.com",
			QueryType: "A",
		}, {
			Engine:    "doh",
			Failure:   nil,
			Hostname:  "example.com",
			QueryType: "AAAA",
		}}
		filler(trace, expect)
		got := trace.FirstDNSLookup()
		if diff := cmp.Diff(got, expect[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestDelayedDNSResponseWithTimeout(t *testing.T) {
	t.Run("OnDelayedDNSResponse saves into the trace", func(t *testing.T) {
		t.Run("when buffer is not full", func(t *testing.T) {
			zeroTime := time.Now()
			td := testingx.NewTimeDeterministic(zeroTime)
			trace := NewTrace(0, zeroTime)
			trace.TimeNowFn = td.Now
			txp := &mocks.DNSTransport{
				MockNetwork: func() string {
					return "udp"
				},
				MockAddress: func() string {
					return "1.1.1.1"
				},
			}
			started := trace.TimeNow()
			query := &mocks.DNSQuery{
				MockType: func() uint16 {
					return dns.TypeA
				},
				MockDomain: func() string {
					return "dns.google.com"
				},
			}
			addrs := []string{"1.1.1.1"}
			finished := trace.TimeNow()
			// 1. fill the trace
			dnsResponse := &mocks.DNSResponse{
				MockDecodeCNAME: func() (string, error) {
					return "", netxlite.ErrOODNSNoAnswer
				},
				MockRcode: func() int {
					return 0
				},
				MockBytes: func() []byte {
					return []byte{}
				},
			}
			err := trace.OnDelayedDNSResponse(started, txp, query, dnsResponse, addrs, nil, finished)
			// 2. read the trace
			got := trace.DelayedDNSResponseWithTimeout(context.Background(), time.Second)
			if err != nil {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 1 {
				t.Fatal("unexpected output from trace")
			}
		})

		t.Run("when buffer is full", func(t *testing.T) {
			zeroTime := time.Now()
			td := testingx.NewTimeDeterministic(zeroTime)
			trace := NewTrace(0, zeroTime)
			trace.TimeNowFn = td.Now
			trace.delayedDNSResponse = make(chan *model.ArchivalDNSLookupResult) // no buffer
			txp := &mocks.DNSTransport{
				MockNetwork: func() string {
					return "udp"
				},
				MockAddress: func() string {
					return "1.1.1.1"
				},
			}
			started := trace.TimeNow()
			query := &mocks.DNSQuery{
				MockType: func() uint16 {
					return dns.TypeA
				},
				MockDomain: func() string {
					return "dns.google.com"
				},
			}
			addrs := []string{"1.1.1.1"}
			finished := trace.TimeNow()
			// 1. attempt to write into the trace
			dnsResponse := &mocks.DNSResponse{
				MockDecodeCNAME: func() (string, error) {
					return "", netxlite.ErrOODNSNoAnswer
				},
				MockRcode: func() int {
					return 0
				},
				MockBytes: func() []byte {
					return []byte{}
				},
			}
			err := trace.OnDelayedDNSResponse(started, txp, query, dnsResponse, addrs, nil, finished)
			if !errors.Is(err, ErrDelayedDNSResponseBufferFull) {
				t.Fatal("unexpected error", err)
			}
			// 2. confirm we didn't write anything
			got := trace.DelayedDNSResponseWithTimeout(context.Background(), time.Second)
			if len(got) != 0 {
				t.Fatal("unexpected output from trace")
			}
		})
	})

	t.Run("DelayedDNSResponseWithTimeout drains the trace", func(t *testing.T) {
		t.Run("context is already cancelled and we still drain the trace", func(t *testing.T) {
			zeroTime := time.Now()
			td := testingx.NewTimeDeterministic(zeroTime)
			trace := NewTrace(0, zeroTime)
			trace.TimeNowFn = td.Now
			txp := &mocks.DNSTransport{
				MockNetwork: func() string {
					return "udp"
				},
				MockAddress: func() string {
					return "1.1.1.1"
				},
			}
			started := trace.TimeNow()
			query := &mocks.DNSQuery{
				MockType: func() uint16 {
					return dns.TypeA
				},
				MockDomain: func() string {
					return "dns.google.com"
				},
			}
			addrs := []string{"1.1.1.1"}
			finished := trace.TimeNow()
			events := 4
			dnsResponse := &mocks.DNSResponse{
				MockDecodeCNAME: func() (string, error) {
					return "", netxlite.ErrOODNSNoAnswer
				},
				MockRcode: func() int {
					return 0
				},
				MockBytes: func() []byte {
					return []byte{}
				},
			}
			for i := 0; i < events; i++ {
				// fill the trace
				trace.delayedDNSResponse <- NewArchivalDNSLookupResultFromRoundTrip(trace.Index, started.Sub(trace.ZeroTime),
					txp, query, dnsResponse, addrs, nil, finished.Sub(trace.ZeroTime))
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // we ensure that the context cancels before draining all the events
			// drain the trace
			got := trace.DelayedDNSResponseWithTimeout(ctx, 10*time.Second)
			if len(got) != 4 {
				t.Fatal("unexpected output from trace", len(got))
			}
		})

		t.Run("normal case where the context times out after we start draining", func(t *testing.T) {
			zeroTime := time.Now()
			td := testingx.NewTimeDeterministic(zeroTime)
			trace := NewTrace(0, zeroTime)
			trace.TimeNowFn = td.Now
			txp := &mocks.DNSTransport{
				MockNetwork: func() string {
					return "udp"
				},
				MockAddress: func() string {
					return "1.1.1.1"
				},
			}
			started := trace.TimeNow()
			query := &mocks.DNSQuery{
				MockType: func() uint16 {
					return dns.TypeA
				},
				MockDomain: func() string {
					return "dns.google.com"
				},
			}
			addrs := []string{"1.1.1.1"}
			finished := trace.TimeNow()
			dnsResponse := &mocks.DNSResponse{
				MockDecodeCNAME: func() (string, error) {
					return "", netxlite.ErrOODNSNoAnswer
				},
				MockRcode: func() int {
					return 0
				},
				MockBytes: func() []byte {
					return []byte{}
				},
			}
			trace.delayedDNSResponse <- NewArchivalDNSLookupResultFromRoundTrip(trace.Index, started.Sub(trace.ZeroTime),
				txp, query, dnsResponse, addrs, nil, finished.Sub(trace.ZeroTime))
			got := trace.DelayedDNSResponseWithTimeout(context.Background(), time.Second)
			if len(got) != 1 {
				t.Fatal("unexpected output from trace")
			}
		})
	})
}

func TestNewArchivalDNSAnswers(t *testing.T) {
	tests := []struct {
		name     string
		addrs    []string
		resp     model.DNSResponse
		expected []model.ArchivalDNSAnswer
	}{{
		name: "with valid input",
		addrs: []string{
			"8.8.4.4",
			"2001:4860:4860::8844",
		},
		resp: nil,
		expected: []model.ArchivalDNSAnswer{{
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "A",
			Hostname:   "",
			IPv4:       "8.8.4.4",
			IPv6:       "",
			TTL:        nil,
		}, {
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "AAAA",
			Hostname:   "",
			IPv4:       "",
			IPv6:       "2001:4860:4860::8844",
			TTL:        nil,
		}},
	}, {
		name: "with invalid IPv4 address",
		addrs: []string{
			"1.1.1.1.1", // invalid because it has five dots
			"2001:4860:4860::8844",
		},
		resp: nil,
		expected: []model.ArchivalDNSAnswer{{
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "AAAA",
			Hostname:   "",
			IPv4:       "",
			IPv6:       "2001:4860:4860::8844",
			TTL:        nil,
		}},
	}, {
		name: "with invalid IPv6 address",
		addrs: []string{
			"8.8.4.4",
			"fe80::a00:20ff:feb9:::4c54", // invalid because it has :::
		},
		resp: nil,
		expected: []model.ArchivalDNSAnswer{{
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "A",
			Hostname:   "",
			IPv4:       "8.8.4.4",
			IPv6:       "",
			TTL:        nil,
		}},
	}, {
		name:     "with empty input",
		addrs:    []string{},
		resp:     nil,
		expected: nil,
	}, {
		name:     "with nil input",
		addrs:    nil,
		resp:     nil,
		expected: nil,
	}, {
		name: "with valid IPv4 address and CNAME",
		addrs: []string{
			"8.8.8.8",
		},
		resp: &mocks.DNSResponse{
			MockDecodeCNAME: func() (string, error) {
				return "dns.google.", nil
			},
		},
		expected: []model.ArchivalDNSAnswer{{
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "A",
			Hostname:   "",
			IPv4:       "8.8.8.8",
			IPv6:       "",
			TTL:        nil,
		}, {
			ASN:        0,
			ASOrgName:  "",
			AnswerType: "CNAME",
			Hostname:   "dns.google.",
			IPv4:       "",
			IPv6:       "",
			TTL:        nil,
		}},
	}, {
		name: "with valid IPv6 address and CNAME",
		addrs: []string{
			"2001:4860:4860::8844",
		},
		resp: &mocks.DNSResponse{
			MockDecodeCNAME: func() (string, error) {
				return "dns.google.", nil
			},
		},
		expected: []model.ArchivalDNSAnswer{{
			ASN:        15169,
			ASOrgName:  "Google LLC",
			AnswerType: "AAAA",
			Hostname:   "",
			IPv4:       "",
			IPv6:       "2001:4860:4860::8844",
			TTL:        nil,
		}, {
			ASN:        0,
			ASOrgName:  "",
			AnswerType: "CNAME",
			Hostname:   "dns.google.",
			IPv4:       "",
			IPv6:       "",
			TTL:        nil,
		}},
	}, {
		name:  "with DecodeCNAME error",
		addrs: []string{},
		resp: &mocks.DNSResponse{
			MockDecodeCNAME: func() (string, error) {
				return "", errors.New("mocked errorr")
			},
		},
		expected: nil,
	}, {
		name:  "with DecodeCNAME success and no CNAME",
		addrs: []string{},
		resp: &mocks.DNSResponse{
			MockDecodeCNAME: func() (string, error) {
				return "", nil
			},
		},
		expected: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newArchivalDNSAnswers(tt.addrs, tt.resp)
			if diff := cmp.Diff(tt.expected, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
