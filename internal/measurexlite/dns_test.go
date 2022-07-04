package measurexlite

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewUnwrappedParallelResolver(t *testing.T) {
	t.Run("NewUnwrappedParallelResolver created an UnwrappedParallelResolver with Trace", func(t *testing.T) {
		underlying := &mocks.Resolver{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NewUnwrappedParallelResolverFn = func(t model.DNSTransport) model.Resolver {
			return underlying
		}
		resolver := trace.NewUnwrappedParallelResolver(&mocks.DNSTransport{})
		resolvert := resolver.(*resolverTrace)
		if resolvert.r != underlying {
			t.Fatal("invalid parallel resolver")
		}
		if resolvert.tx != trace {
			t.Fatal("invalid trace")
		}
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
		resolver := trace.NewUnwrappedParallelResolver(txp)
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
		resolver := trace.NewUnwrappedParallelResolver(txp)
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
