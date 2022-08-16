package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestParallelResolver(t *testing.T) {
	t.Run("transport okay", func(t *testing.T) {
		txp := NewUnwrappedDNSOverTLSTransport((&tls.Dialer{}).DialContext, "8.8.8.8:853")
		r := NewUnwrappedParallelResolver(txp)
		rtx := r.Transport()
		if rtx.Network() != "dot" || rtx.Address() != "8.8.8.8:853" {
			t.Fatal("not the transport we expected")
		}
		if r.Network() != rtx.Network() {
			t.Fatal("invalid network seen from the resolver")
		}
		if r.Address() != rtx.Address() {
			t.Fatal("invalid address seen from the resolver")
		}
	})

	t.Run("LookupHost", func(t *testing.T) {
		t.Run("RoundTrip error", func(t *testing.T) {
			mocked := errors.New("mocked error")
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, mocked
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewUnwrappedParallelResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
		})

		t.Run("empty reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							return nil, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewUnwrappedParallelResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
		})

		t.Run("with A reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							if query.Type() != dns.TypeA {
								return nil, nil
							}
							return []string{"8.8.8.8"}, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewUnwrappedParallelResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("not the result we expected", addrs)
			}
		})

		t.Run("with AAAA reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							if query.Type() != dns.TypeAAAA {
								return nil, nil
							}
							return []string{"::1"}, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewUnwrappedParallelResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "::1" {
				t.Fatal("not the result we expected", addrs)
			}
		})

		t.Run("A failure takes precedence over AAAA failure", func(t *testing.T) {
			afailure := errors.New("a failure")
			aaaafailure := errors.New("aaaa failure")
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					switch query.Type() {
					case dns.TypeA:
						return nil, afailure
					case dns.TypeAAAA:
						return nil, aaaafailure
					default:
						return nil, errors.New("unexpected query")
					}
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewUnwrappedParallelResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, afailure) {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 0 {
				t.Fatal("not the result we expected")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		r := &ParallelResolver{
			Txp: &mocks.DNSTransport{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		r.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("for round-trip error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &ParallelResolver{
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						return nil, expected
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})

		t.Run("for DecodeHTTPS error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &ParallelResolver{
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						response := &mocks.DNSResponse{
							MockDecodeHTTPS: func() (*model.HTTPSSvc, error) {
								return nil, expected
							},
						}
						return response, nil
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("for round-trip error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &ParallelResolver{
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						return nil, expected
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if ns != nil {
				t.Fatal("unexpected result")
			}
		})

		t.Run("for decode error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &ParallelResolver{
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						response := &mocks.DNSResponse{
							MockDecodeNS: func() ([]*net.NS, error) {
								return nil, expected
							},
						}
						return response, nil
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupNS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})
	})

	t.Run("uses a context-injected custom trace (success case)", func(t *testing.T) {
		var (
			onLookupACalled        bool
			onLookupAAAACalled     bool
			goodQueryTypeA         bool
			goodQueryTypeAAAA      bool
			goodLookupAddrsA       bool
			goodLookupAddrsAAAA    bool
			goodLookupErrorA       bool
			goodLookupErrorAAAA    bool
			goodLookupResolverA    bool
			goodLookupResolverAAAA bool
		)
		expectedA := []string{"1.1.1.1"}
		expectedAAAA := []string{"::1"}
		txp := &mocks.DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				if query.Type() == dns.TypeA {
					return &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							return expectedA, nil
						},
					}, nil
				}
				if query.Type() == dns.TypeAAAA {
					return &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							return expectedAAAA, nil
						},
					}, nil
				}
				return nil, errors.New("unexpected query type")
			},
			MockNetwork: func() string {
				return "mocked"
			},
			MockRequiresPadding: func() bool {
				return false
			},
		}
		r := NewUnwrappedParallelResolver(txp)
		zeroTime := time.Now()
		deteterministicTime := testingx.NewTimeDeterministic(zeroTime)
		tx := &mocks.Trace{
			MockTimeNow: deteterministicTime.Now,
			MockOnDNSRoundTripForLookupHost: func(started time.Time, reso model.Resolver, query model.DNSQuery,
				response model.DNSResponse, addrs []string, err error, finished time.Time) {
				if query.Type() == dns.TypeA {
					onLookupACalled = true
					goodQueryTypeA = (query.Type() == dns.TypeA)
					goodLookupAddrsA = (cmp.Diff(expectedA, addrs) == "")
					goodLookupErrorA = (err == nil)
					goodLookupResolverA = (reso.Network() == "mocked")
				}
				if query.Type() == dns.TypeAAAA {
					onLookupAAAACalled = true
					goodQueryTypeAAAA = (query.Type() == dns.TypeAAAA)
					goodLookupAddrsAAAA = (cmp.Diff(expectedAAAA, addrs) == "")
					goodLookupErrorAAAA = (err == nil)
					goodLookupResolverAAAA = (reso.Network() == "mocked")
				}
			},
		}
		want := []string{"1.1.1.1", "::1"}
		ctx := ContextWithTrace(context.Background(), tx)
		addrs, err := r.LookupHost(ctx, "example.com")
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		// Note: the implementation always puts IPv4 addrs before IPv6 addrs
		if diff := cmp.Diff(want, addrs); diff != "" {
			t.Fatal("unexpected addresses")
		}

		t.Run("with A reply", func(t *testing.T) {
			if !onLookupACalled {
				t.Fatal("onLookupACalled not called")
			}
			if !goodQueryTypeA {
				t.Fatal("unexpected query type in parallel resolver")
			}
			if !goodLookupAddrsA {
				t.Fatal("unexpected addresses in LookupHost")
			}
			if !goodLookupErrorA {
				t.Fatal("unexpected error in trace")
			}
			if !goodLookupResolverA {
				t.Fatal("unexpected resolver network encountered")
			}
		})

		t.Run("with AAAA reply", func(t *testing.T) {
			if !onLookupAAAACalled {
				t.Fatal("onLookupAAAACalled not called")
			}
			if !goodQueryTypeAAAA {
				t.Fatal("unexpected query type in parallel resolver")
			}
			if !goodLookupAddrsAAAA {
				t.Fatal("unexpected addresses in LookupHost")
			}
			if !goodLookupErrorAAAA {
				t.Fatal("unexpected error in trace")
			}
			if !goodLookupResolverAAAA {
				t.Fatal("unexpected resolver network encountered")
			}
		})
	})

	t.Run("uses a context-injected custom trace (failure case)", func(t *testing.T) {
		var (
			onLookupACalled        bool
			onLookupAAAACalled     bool
			goodQueryTypeA         bool
			goodQueryTypeAAAA      bool
			goodLookupAddrsA       bool
			goodLookupAddrsAAAA    bool
			goodLookupErrorA       bool
			goodLookupErrorAAAA    bool
			goodLookupResolverA    bool
			goodLookupResolverAAAA bool
		)
		expected := errors.New("mocked")
		txp := &mocks.DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				if query.Type() == dns.TypeAAAA || query.Type() == dns.TypeA {
					return nil, expected
				}
				return nil, errors.New("unexpected query type")
			},
			MockNetwork: func() string {
				return "mocked"
			},
			MockRequiresPadding: func() bool {
				return false
			},
		}
		r := NewUnwrappedParallelResolver(txp)
		zeroTime := time.Now()
		deteterministicTime := testingx.NewTimeDeterministic(zeroTime)
		tx := &mocks.Trace{
			MockTimeNow: deteterministicTime.Now,
			MockOnDNSRoundTripForLookupHost: func(started time.Time, reso model.Resolver, query model.DNSQuery,
				response model.DNSResponse, addrs []string, err error, finished time.Time) {
				if query.Type() == dns.TypeA {
					onLookupACalled = true
					goodQueryTypeA = (query.Type() == dns.TypeA)
					goodLookupAddrsA = (len(addrs) == 0)
					goodLookupErrorA = errors.Is(expected, err)
					goodLookupResolverA = (reso.Network() == "mocked")
					return
				}
				if query.Type() == dns.TypeAAAA {
					onLookupAAAACalled = true
					goodQueryTypeAAAA = (query.Type() == dns.TypeAAAA)
					goodLookupAddrsAAAA = (len(addrs) == 0)
					goodLookupErrorAAAA = errors.Is(expected, err)
					goodLookupResolverAAAA = (reso.Network() == "mocked")
					return
				}
			},
		}
		ctx := ContextWithTrace(context.Background(), tx)
		addrs, err := r.LookupHost(ctx, "example.com")
		if !errors.Is(expected, err) {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 0 {
			t.Fatal("unexpected addresses")
		}

		t.Run("with A reply", func(t *testing.T) {
			if !onLookupACalled {
				t.Fatal("onLookupACalled not called")
			}
			if !goodQueryTypeA {
				t.Fatal("unexpected query type in parallel resolver")
			}
			if !goodLookupAddrsA {
				t.Fatal("unexpected addresses in LookupHost")
			}
			if !goodLookupErrorA {
				t.Fatal("unexpected error in trace")
			}
			if !goodLookupResolverA {
				t.Fatal("unexpected resolver network encountered")
			}
		})

		t.Run("with AAAA reply", func(t *testing.T) {
			if !onLookupAAAACalled {
				t.Fatal("onLookupAAAACalled not called")
			}
			if !goodQueryTypeAAAA {
				t.Fatal("unexpected query type in parallel resolver")
			}
			if !goodLookupAddrsAAAA {
				t.Fatal("unexpected addresses in LookupHost")
			}
			if !goodLookupErrorAAAA {
				t.Fatal("unexpected error in trace")
			}
			if !goodLookupResolverAAAA {
				t.Fatal("unexpected resolver network encountered")
			}
		})
	})
}
