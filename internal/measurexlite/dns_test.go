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
				t.Fatal("unexpected DNS events")
			}
			for _, ev := range events {
				if ev.ResolverAddress != "dns.google" {
					t.Fatal("unexpected resolver address")
				}
				if ev.Engine != "mocked" {
					t.Fatal("unexpected engine")
				}
				if len(ev.Answers) != 1 {
					t.Fatal("expected single answer in DNSLookup event")
				}
				if ev.QueryType == "A" && ev.Answers[0].IPv4 != "1.1.1.1" {
					t.Fatal("unexpected A query result")
				}
				if ev.QueryType == "AAAA" && ev.Answers[0].IPv6 != "fe80::a00:20ff:feb9:4c54" {
					t.Fatal("unexpected AAAA query result")
				}
			}
		})
	})

	t.Run("LookupHost discards events when buffers are full", func(t *testing.T) {
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.dnsLookup = make(chan *model.ArchivalDNSLookupResult) // no buffer
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
		if resolver.Network() != "system" {
			t.Fatal("unexpected resolver network")
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
	t.Run("OnDelayedDNSResponseWithTimeout saves into the trace", func(t *testing.T) {
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
			err := trace.OnDelayedDNSResponse(started, txp, query, &mocks.DNSResponse{},
				addrs, nil, finished)
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
			err := trace.OnDelayedDNSResponse(started, txp, query, &mocks.DNSResponse{},
				addrs, nil, finished)
			got := trace.DelayedDNSResponseWithTimeout(context.Background(), time.Second)
			if !errors.Is(err, ErrDelayedDNSResponseBufferFull) {
				t.Fatal("unexpected error", err)
			}
			if len(got) != 0 {
				t.Fatal("unexpected output from trace")
			}
		})
	})

	t.Run("DelayedDNSResponseWithTimeout drains the trace", func(t *testing.T) {
		t.Run("context times out", func(t *testing.T) {
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
			for i := 0; i < events; i++ {
				trace.delayedDNSResponse <- NewArchivalDNSLookupResultFromRoundTrip(trace.Index, started.Sub(trace.ZeroTime),
					txp, query, &mocks.DNSResponse{}, addrs, nil, finished.Sub(trace.ZeroTime))
			}
			// we ensure that the context cancels before draining all the events
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			got := trace.DelayedDNSResponseWithTimeout(ctx, 10*time.Second)
			if len(got) >= 4 {
				t.Fatal("unexpected output from trace")
			}
		})

		t.Run("context does not time out", func(t *testing.T) {
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
			trace.delayedDNSResponse <- NewArchivalDNSLookupResultFromRoundTrip(trace.Index, started.Sub(trace.ZeroTime),
				txp, query, &mocks.DNSResponse{}, addrs, nil, finished.Sub(trace.ZeroTime))
			got := trace.DelayedDNSResponseWithTimeout(context.Background(), time.Second)
			if len(got) != 1 {
				t.Fatal("unexpected output from trace")
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
