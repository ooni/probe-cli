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

func TestNewUnwrappedParallelResolver(t *testing.T) {
	t.Run("NewUnwrappedParallelResolver creates an UnwrappedParallelResolver with Trace", func(t *testing.T) {
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

		t.Run("DNSLookup Events", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip()
			if len(events) != 2 {
				t.Fatal("expected to see single DNSLookup event")
			}
			for i, ev := range events {
				if ev.Failure != nil {
					t.Fatal("unexpected err", *(ev.Failure))
				}
				if ev.ResolverAddress != "dns.google" {
					t.Fatal("unexpected address field")
				}
				answer := ev.Answers[0]
				// checking order of results
				if i == 0 {
					if answer.AnswerType != "A" || answer.IPv4 != "1.1.1.1" {
						t.Fatal("unexpected DNS answer", answer)
					}
				}
				if i == 1 {
					if answer.AnswerType != "AAAA" || answer.IPv6 != "fe80::a00:20ff:feb9:4c54" {
						t.Fatal("unexpected DNS answer", answer)
					}
				}
			}
		})
	})

	t.Run("LookupHost discards events when buffers are full", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.DNSLookup = map[uint16]chan *model.ArchivalDNSLookupResult{
			dns.TypeA:    make(chan *model.ArchivalDNSLookupResult),
			dns.TypeAAAA: make(chan *model.ArchivalDNSLookupResult),
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

		t.Run("DNSLookup events", func(t *testing.T) {
			events := trace.DNSLookupsFromRoundTrip()
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
