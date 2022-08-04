package measurexlite

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewParallelResolver(t *testing.T) {
	t.Run("NewParallelResolverTrace creates an ParallelResolver with Trace", func(t *testing.T) {
		underlying := &mocks.Resolver{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewParallelResolverFn = func() model.Resolver {
			return underlying
		}
		resolver := trace.newParallelResolverTrace(func() model.Resolver {
			return nil
		})
		resolvert := resolver.(*resolverTrace)
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
		newMockResolver := func() model.Resolver {
			return &mocks.Resolver{
				MockAddress: func() string {
					return "dns.google"
				},
				MockNetwork: func() string {
					return "udp"
				},
				MockCloseIdleConnections: func() {
					called = true
				},
			}
		}
		resolver := trace.newParallelResolver(newMockResolver)

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
		newResolver := func() model.Resolver {
			return netxlite.NewUnwrappedParallelResolver(txp)
		}
		resolver := trace.newParallelResolverTrace(newResolver)
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

		t.Run("DNSLookups QueryType A", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeA)
			if len(events) != 1 {
				t.Fatal("expected to see single DNSLookup event")
			}
			lookup := events[0]
			answers := lookup.Answers
			if lookup.Failure != nil {
				t.Fatal("unexpected err", *(lookup.Failure))
			}
			if lookup.ResolverAddress != "dns.google" {
				t.Fatal("unexpected address field")
			}
			if len(answers) != 1 {
				t.Fatal("expected 1 DNS answer, got", len(answers))
			}
			if answers[0].AnswerType != "A" || answers[0].IPv4 != "1.1.1.1" {
				t.Fatal("unexpected DNS answer", answers)
			}
		})

		t.Run("DNSLookups QueryType AAAA", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeAAAA)
			if len(events) != 1 {
				t.Fatal("expected to see single DNSLookup event")
			}
			lookup := events[0]
			answers := lookup.Answers
			if lookup.Failure != nil {
				t.Fatal("unexpected err", *(lookup.Failure))
			}
			if lookup.ResolverAddress != "dns.google" {
				t.Fatal("unexpected address field")
			}
			if len(answers) != 1 {
				t.Fatal("expected 1 DNS answer, got", len(answers))
			}
			if answers[0].AnswerType != "AAAA" || answers[0].IPv6 != "fe80::a00:20ff:feb9:4c54" {
				t.Fatal("unexpected DNS answer", answers)
			}
		})
	})

	t.Run("LookupHost discards events when buffers are full", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.DNSLookup = map[uint16]chan *model.ArchivalDNSLookupResult{
			dns.TypeA:    make(chan *model.ArchivalDNSLookupResult), // no buffer
			dns.TypeAAAA: make(chan *model.ArchivalDNSLookupResult), // no buffer
		}
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
		newResolver := func() model.Resolver {
			return netxlite.NewUnwrappedParallelResolver(txp)
		}
		resolver := trace.newParallelResolverTrace(newResolver)
		ctx := context.Background()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 2 {
			t.Fatal("unexpected array output", addrs)
		}

		t.Run("DNSLookups QueryType A", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeA)
			if len(events) != 0 {
				t.Fatal("expected to see no DNSLookup")
			}
		})
		t.Run("DNSLookups QueryType AAAA", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeAAAA)
			if len(events) != 0 {
				t.Fatal("expected to see no DNSLookup")
			}
		})
	})
}

func TestNewSimpleResolver(t *testing.T) {
	t.Run("NewSimpleResolverTrace creates a SimpleResolver with Trace", func(t *testing.T) {
		underlying := &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return []string{}, nil
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewSimpleResolverFn = func() model.SimpleResolver {
			return underlying
		}
		resolver := trace.newSimpleResolverTrace(func() model.SimpleResolver {
			return nil
		})
		resolvert := resolver.(*simpleResolverTrace)
		if resolvert.r != underlying {
			t.Fatal("invalid simple resolver")
		}
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("Trace-aware resolver forwards underlying functions", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		newMockResolver := func() model.SimpleResolver {
			return &mocks.Resolver{
				MockNetwork: func() string {
					return "udp"
				},
			}
		}
		resolver := trace.newSimpleResolver(newMockResolver)

		t.Run("Network is correctly forwarded", func(t *testing.T) {
			got := resolver.Network()
			if got != "udp" {
				t.Fatal("Network not called")
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
		newSimpleResolver := func() model.SimpleResolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					reso := netxlite.NewUnwrappedParallelResolver(txp)
					return reso.LookupHost(ctx, domain)
				},
			}
		}
		resolver := trace.newSimpleResolverTrace(newSimpleResolver)
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

		t.Run("DNSLookups QueryType A", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeA)
			if len(events) != 1 {
				t.Fatal("expected to see single DNSLookup event")
			}
			lookup := events[0]
			answers := lookup.Answers
			if lookup.Failure != nil {
				t.Fatal("unexpected err", *(lookup.Failure))
			}
			if lookup.ResolverAddress != "dns.google" {
				t.Fatal("unexpected address field")
			}
			if len(answers) != 1 {
				t.Fatal("expected 1 DNS answer, got", len(answers))
			}
			if answers[0].AnswerType != "A" || answers[0].IPv4 != "1.1.1.1" {
				t.Fatal("unexpected DNS answer", answers)
			}
		})

		t.Run("DNSLookups QueryType AAAA", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeAAAA)
			if len(events) != 1 {
				t.Fatal("expected to see single DNSLookup event")
			}
			lookup := events[0]
			answers := lookup.Answers
			if lookup.Failure != nil {
				t.Fatal("unexpected err", *(lookup.Failure))
			}
			if lookup.ResolverAddress != "dns.google" {
				t.Fatal("unexpected address field")
			}
			if len(answers) != 1 {
				t.Fatal("expected 1 DNS answer, got", len(answers))
			}
			if answers[0].AnswerType != "AAAA" || answers[0].IPv6 != "fe80::a00:20ff:feb9:4c54" {
				t.Fatal("unexpected DNS answer", answers)
			}
		})
	})

	t.Run("LookupHost discards events when buffers are full", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.DNSLookup = map[uint16]chan *model.ArchivalDNSLookupResult{
			dns.TypeA:    make(chan *model.ArchivalDNSLookupResult), // no buffer
			dns.TypeAAAA: make(chan *model.ArchivalDNSLookupResult), // no buffer
		}
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
		newSimpleResolver := func() model.SimpleResolver {
			return &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					reso := netxlite.NewUnwrappedParallelResolver(txp)
					return reso.LookupHost(ctx, domain)
				},
			}
		}
		resolver := trace.newSimpleResolverTrace(newSimpleResolver)
		ctx := context.Background()
		addrs, err := resolver.LookupHost(ctx, "example.com")
		if err != nil {
			t.Fatal("unexpected err", err)
		}
		if len(addrs) != 2 {
			t.Fatal("unexpected array output", addrs)
		}

		t.Run("DNSLookups QueryType A", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeA)
			if len(events) != 0 {
				t.Fatal("expected to see no DNSLookup")
			}
		})
		t.Run("DNSLookups QueryType AAAA", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip(dns.TypeAAAA)
			if len(events) != 0 {
				t.Fatal("expected to see no DNSLookup")
			}
		})
	})
}

func TestAnswersFromAddrs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{{
		name: "with valid input",
		args: []string{"1.1.1.1", "fe80::a00:20ff:feb9:4c54"},
	}, {
		name: "with invalid IPv4 address",
		args: []string{"1.1.1.1.1", "fe80::a00:20ff:feb9:4c54"},
	}, {
		name: "with invalid IPv6 address",
		args: []string{"1.1.1.1", "fe80::a00:20ff:feb9:::4c54"},
	}, {
		name: "with empty input",
		args: []string{},
	}, {
		name: "with nil input",
		args: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := archivalAnswersFromAddrs(tt.args)
			var idx int
			for _, inp := range tt.args {
				ip6, err := netxlite.IsIPv6(inp)
				if err != nil {
					continue
				}
				if idx >= len(got) {
					t.Fatal("unexpected array length")
				}
				answer := got[idx]
				if ip6 {
					if answer.AnswerType != "AAAA" || answer.IPv6 != inp {
						t.Fatal("unexpected output", answer)
					}
				} else {
					if answer.AnswerType != "A" || answer.IPv4 != inp {
						t.Fatal("unexpected output", answer)
					}
				}
				idx++
			}
			if idx != len(got) {
				t.Fatal("unexpected array length", len(got))
			}
		})
	}
}

func TestDNSLookupsFromRoundTrips(t *testing.T) {
	zeroTime := time.Now()
	trace := NewTrace(0, zeroTime)
	checkPanic := func(query uint16, f func(uint16) []*model.ArchivalDNSLookupResult) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatal("unexpected panic encoutered")
			}
		}()
		f(query)
	}
	t.Run("DNSLookup is nil", func(t *testing.T) {
		trace.DNSLookup = nil
		checkPanic(dns.TypeA, trace.DNSLookupsFromRoundTrip)
	})
	t.Run("Query has nil channel", func(t *testing.T) {
		trace.DNSLookup = map[uint16]chan *model.ArchivalDNSLookupResult{
			dns.TypeA: nil,
		}
		checkPanic(dns.TypeA, trace.DNSLookupsFromRoundTrip)
	})
}
