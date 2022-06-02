package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewResolverSystem(t *testing.T) {
	resolver := NewResolverStdlib(log.Log)
	idna := resolver.(*resolverIDNA)
	logger := idna.Resolver.(*resolverLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	shortCircuit := logger.Resolver.(*resolverShortCircuitIPAddr)
	errWrapper := shortCircuit.Resolver.(*resolverErrWrapper)
	reso := errWrapper.Resolver.(*resolverSystem)
	txpErrWrapper := reso.t.(*dnsTransportErrWrapper)
	_ = txpErrWrapper.DNSTransport.(*dnsOverGetaddrinfoTransport)
}

func TestNewSerialResolverUDP(t *testing.T) {
	d := NewDialerWithoutResolver(log.Log)
	resolver := NewSerialResolverUDP(log.Log, d, "1.1.1.1:53")
	idna := resolver.(*resolverIDNA)
	logger := idna.Resolver.(*resolverLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	shortCircuit := logger.Resolver.(*resolverShortCircuitIPAddr)
	errWrapper := shortCircuit.Resolver.(*resolverErrWrapper)
	serio := errWrapper.Resolver.(*SerialResolver)
	txp := serio.Transport().(*dnsTransportErrWrapper)
	dnsTxp := txp.DNSTransport.(*DNSOverUDPTransport)
	if dnsTxp.Address() != "1.1.1.1:53" {
		t.Fatal("invalid address")
	}
}

func TestNewParallelResolverUDP(t *testing.T) {
	d := NewDialerWithoutResolver(log.Log)
	resolver := NewParallelResolverUDP(log.Log, d, "1.1.1.1:53")
	idna := resolver.(*resolverIDNA)
	logger := idna.Resolver.(*resolverLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	shortCircuit := logger.Resolver.(*resolverShortCircuitIPAddr)
	errWrapper := shortCircuit.Resolver.(*resolverErrWrapper)
	para := errWrapper.Resolver.(*ParallelResolver)
	txp := para.Transport().(*dnsTransportErrWrapper)
	dnsTxp := txp.DNSTransport.(*DNSOverUDPTransport)
	if dnsTxp.Address() != "1.1.1.1:53" {
		t.Fatal("invalid address")
	}
}

func TestNewParallelDNSOverHTTPSResolver(t *testing.T) {
	resolver := NewParallelDNSOverHTTPSResolver(log.Log, "https://1.1.1.1/dns-query")
	idna := resolver.(*resolverIDNA)
	logger := idna.Resolver.(*resolverLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	shortCircuit := logger.Resolver.(*resolverShortCircuitIPAddr)
	errWrapper := shortCircuit.Resolver.(*resolverErrWrapper)
	para := errWrapper.Resolver.(*ParallelResolver)
	txp := para.Transport().(*dnsTransportErrWrapper)
	dnsTxp := txp.DNSTransport.(*DNSOverHTTPSTransport)
	if dnsTxp.Address() != "https://1.1.1.1/dns-query" {
		t.Fatal("invalid address")
	}
}

func TestResolverSystem(t *testing.T) {
	t.Run("Network", func(t *testing.T) {
		expected := "antani"
		r := &resolverSystem{
			t: &mocks.DNSTransport{
				MockNetwork: func() string {
					return expected
				},
			},
		}
		if r.Network() != expected {
			t.Fatal("invalid Network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		expected := "address"
		r := &resolverSystem{
			t: &mocks.DNSTransport{
				MockAddress: func() string {
					return expected
				},
			},
		}
		if r.Address() != expected {
			t.Fatal("invalid Address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		r := &resolverSystem{
			t: &mocks.DNSTransport{
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

	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with success", func(t *testing.T) {
			expected := []string{"8.8.8.8", "8.8.4.4"}
			r := &resolverSystem{
				t: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						if query.Type() != dns.TypeANY {
							return nil, errors.New("unexpected lookup type")
						}
						resp := &mocks.DNSResponse{
							MockDecodeLookupHost: func() ([]string, error) {
								return expected, nil
							},
						}
						return resp, nil
					},
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "dns.google")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, addrs); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with failure", func(t *testing.T) {
			expected := errors.New("mocked")
			r := &resolverSystem{
				t: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						if query.Type() != dns.TypeANY {
							return nil, errors.New("unexpected lookup type")
						}
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) != 0 {
				t.Fatal("invalid addrs")
			}
		})
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		r := &resolverSystem{}
		https, err := r.LookupHTTPS(context.Background(), "x.org")
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("not the error we expected")
		}
		if https != nil {
			t.Fatal("expected nil result")
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		r := &resolverSystem{}
		ns, err := r.LookupNS(context.Background(), "x.org")
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("not the error we expected")
		}
		if len(ns) != 0 {
			t.Fatal("expected no results")
		}
	})
}

func TestResolverLogger(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := []string{"1.1.1.1"}
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return expected, nil
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			addrs, err := r.LookupHost(context.Background(), "dns.google")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, addrs); diff != "" {
				t.Fatal(diff)
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})

		t.Run("with failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := errors.New("mocked error")
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			addrs, err := r.LookupHost(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("expected nil addr here")
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("with success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := &model.HTTPSSvc{
				ALPN: []string{"h3"},
				IPv4: []string{"1.1.1.1"},
			}
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return expected, nil
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			https, err := r.LookupHTTPS(context.Background(), "dns.google")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, https); diff != "" {
				t.Fatal(diff)
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})

		t.Run("with failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := errors.New("mocked error")
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return nil, expected
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			https, err := r.LookupHTTPS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if https != nil {
				t.Fatal("expected nil addr here")
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		reso := &resolverLogger{
			Resolver: child,
			Logger:   model.DiscardLogger,
		}
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("with success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := []*net.NS{{
				Host: "ns1.zdns.google.",
			}}
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return expected, nil
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			ns, err := r.LookupNS(context.Background(), "dns.google")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, ns); diff != "" {
				t.Fatal(diff)
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})

		t.Run("with failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			expected := errors.New("mocked error")
			r := &resolverLogger{
				Logger: lo,
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return nil, expected
					},
					MockNetwork: func() string {
						return "system"
					},
					MockAddress: func() string {
						return ""
					},
				},
			}
			ns, err := r.LookupNS(context.Background(), "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
			if ns != nil {
				t.Fatal("expected nil addr here")
			}
			if count != 2 {
				t.Fatal("unexpected count")
			}
		})
	})
}

func TestResolverIDNA(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with valid IDNA in input", func(t *testing.T) {
			expectedIPs := []string{"77.88.55.66"}
			r := &resolverIDNA{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						if domain != "xn--d1acpjx3f.xn--p1ai" {
							return nil, errors.New("passed invalid domain")
						}
						return expectedIPs, nil
					},
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "яндекс.рф")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expectedIPs, addrs); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with invalid punycode", func(t *testing.T) {
			r := &resolverIDNA{Resolver: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, errors.New("should not happen")
				},
			}}
			// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "xn--0000h")
			if err == nil || !strings.HasPrefix(err.Error(), "idna: invalid label") {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected no response here")
			}
		})
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("with valid IDNA in input", func(t *testing.T) {
			expected := &model.HTTPSSvc{
				ALPN: []string{"h3"},
				IPv4: []string{"1.1.1.1"},
				IPv6: []string{},
			}
			r := &resolverIDNA{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						if domain != "xn--d1acpjx3f.xn--p1ai" {
							return nil, errors.New("passed invalid domain")
						}
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "яндекс.рф")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, https); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with invalid punycode", func(t *testing.T) {
			r := &resolverIDNA{Resolver: &mocks.Resolver{
				MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
					return nil, errors.New("should not happen")
				},
			}}
			// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "xn--0000h")
			if err == nil || !strings.HasPrefix(err.Error(), "idna: invalid label") {
				t.Fatal("not the error we expected")
			}
			if https != nil {
				t.Fatal("expected no response here")
			}
		})
	})

	t.Run("Network", func(t *testing.T) {
		child := &mocks.Resolver{
			MockNetwork: func() string {
				return "x"
			},
		}
		r := &resolverIDNA{child}
		if r.Network() != "x" {
			t.Fatal("invalid network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		child := &mocks.Resolver{
			MockAddress: func() string {
				return "x"
			},
		}
		r := &resolverIDNA{child}
		if r.Address() != "x" {
			t.Fatal("invalid address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		reso := &resolverIDNA{child}
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("with valid IDNA in input", func(t *testing.T) {
			expected := []*net.NS{{
				Host: "ns1.zdns.google.",
			}}
			r := &resolverIDNA{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						if domain != "xn--d1acpjx3f.xn--p1ai" {
							return nil, errors.New("passed invalid domain")
						}
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "яндекс.рф")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, ns); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with invalid punycode", func(t *testing.T) {
			r := &resolverIDNA{Resolver: &mocks.Resolver{
				MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
					return nil, errors.New("should not happen")
				},
			}}
			// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "xn--0000h")
			if err == nil || !strings.HasPrefix(err.Error(), "idna: invalid label") {
				t.Fatal("not the error we expected")
			}
			if ns != nil {
				t.Fatal("expected no response here")
			}
		})
	})
}

func TestResolverShortCircuitIPAddr(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with IP addr", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "8.8.8.8")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("invalid result")
			}
		})

		t.Run("with domain", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "dns.google")
			if err == nil || err.Error() != "mocked error" {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("invalid result")
			}
		})
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("with IPv4 addr", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "8.8.8.8")
			if err != nil {
				t.Fatal(err)
			}
			if len(https.IPv4) != 1 || https.IPv4[0] != "8.8.8.8" {
				t.Fatal("invalid result")
			}
		})

		t.Run("with IPv6 addr", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "::1")
			if err != nil {
				t.Fatal(err)
			}
			if len(https.IPv6) != 1 || https.IPv6[0] != "::1" {
				t.Fatal("invalid result")
			}
		})

		t.Run("with domain", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "dns.google")
			if err == nil || err.Error() != "mocked error" {
				t.Fatal("not the error we expected", err)
			}
			if https != nil {
				t.Fatal("invalid result")
			}
		})
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("with IPv4 addr", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "8.8.8.8")
			if !errors.Is(err, ErrDNSIPAddress) {
				t.Fatal("unexpected error", err)
			}
			if len(ns) > 0 {
				t.Fatal("invalid result")
			}
		})

		t.Run("with IPv6 addr", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "::1")
			if !errors.Is(err, ErrDNSIPAddress) {
				t.Fatal("unexpected error", err)
			}
			if len(ns) > 0 {
				t.Fatal("invalid result")
			}
		})

		t.Run("with domain", func(t *testing.T) {
			r := &resolverShortCircuitIPAddr{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return nil, errors.New("mocked error")
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "dns.google")
			if err == nil || err.Error() != "mocked error" {
				t.Fatal("not the error we expected", err)
			}
			if len(ns) > 0 {
				t.Fatal("invalid result")
			}
		})
	})

	t.Run("Network", func(t *testing.T) {
		child := &mocks.Resolver{
			MockNetwork: func() string {
				return "x"
			},
		}
		reso := &resolverShortCircuitIPAddr{child}
		if reso.Network() != "x" {
			t.Fatal("invalid result")
		}
	})

	t.Run("Address", func(t *testing.T) {
		child := &mocks.Resolver{
			MockAddress: func() string {
				return "x"
			},
		}
		reso := &resolverShortCircuitIPAddr{child}
		if reso.Address() != "x" {
			t.Fatal("invalid result")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		reso := &resolverShortCircuitIPAddr{child}
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestIsIPv6(t *testing.T) {
	t.Run("with neither IPv4 nor IPv6 as input", func(t *testing.T) {
		ipv6, err := IsIPv6("example.com")
		if !errors.Is(err, ErrInvalidIP) {
			t.Fatal("not the error we expected", err)
		}
		if ipv6 {
			t.Fatal("expected false")
		}
	})

	t.Run("with IPv4 as input", func(t *testing.T) {
		ipv6, err := IsIPv6("1.2.3.4")
		if err != nil {
			t.Fatal(err)
		}
		if ipv6 {
			t.Fatal("expected false")
		}
	})

	t.Run("with IPv6 as input", func(t *testing.T) {
		ipv6, err := IsIPv6("::1")
		if err != nil {
			t.Fatal(err)
		}
		if !ipv6 {
			t.Fatal("expected true")
		}
	})
}

func TestNullResolver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		r := &NullResolver{}
		ctx := context.Background()
		addrs, err := r.LookupHost(ctx, "dns.google")
		if !errors.Is(err, ErrNoResolver) {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil addr")
		}
		if r.Network() != "null" {
			t.Fatal("invalid network")
		}
		if r.Address() != "" {
			t.Fatal("invalid address")
		}
		r.CloseIdleConnections() // for coverage
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		r := &NullResolver{}
		ctx := context.Background()
		addrs, err := r.LookupHTTPS(ctx, "dns.google")
		if !errors.Is(err, ErrNoResolver) {
			t.Fatal("not the error we expected", err)
		}
		if addrs != nil {
			t.Fatal("expected nil addr")
		}
		if r.Network() != "null" {
			t.Fatal("invalid network")
		}
		if r.Address() != "" {
			t.Fatal("invalid address")
		}
		r.CloseIdleConnections() // for coverage
	})

	t.Run("LookupNS", func(t *testing.T) {
		r := &NullResolver{}
		ctx := context.Background()
		ns, err := r.LookupNS(ctx, "dns.google")
		if !errors.Is(err, ErrNoResolver) {
			t.Fatal("unexpected error", err)
		}
		if len(ns) > 0 {
			t.Fatal("unexpected result")
		}
	})
}

func TestResolverErrWrapper(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expected := []string{"8.8.8.8", "8.8.4.4"}
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, addrs); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := io.EOF
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "")
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if addrs != nil {
				t.Fatal("unexpected addrs")
			}
		})
	})

	t.Run("Network", func(t *testing.T) {
		expected := "foobar"
		reso := &resolverErrWrapper{
			Resolver: &mocks.Resolver{
				MockNetwork: func() string {
					return expected
				},
			},
		}
		if reso.Network() != expected {
			t.Fatal("invalid network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		expected := "foobar"
		reso := &resolverErrWrapper{
			Resolver: &mocks.Resolver{
				MockAddress: func() string {
					return expected
				},
			},
		}
		if reso.Address() != expected {
			t.Fatal("invalid address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		reso := &resolverErrWrapper{
			Resolver: &mocks.Resolver{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expected := &model.HTTPSSvc{
				ALPN: []string{"h3"},
			}
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			https, err := reso.LookupHTTPS(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, https); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := io.EOF
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			https, err := reso.LookupHTTPS(ctx, "")
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected addrs")
			}
		})
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expected := []*net.NS{{
				Host: "antani.local.",
			}}
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			ns, err := reso.LookupNS(ctx, "antani.local")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, ns); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := io.EOF
			reso := &resolverErrWrapper{
				Resolver: &mocks.Resolver{
					MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			ns, err := reso.LookupNS(ctx, "")
			if err == nil || err.Error() != FailureEOFError {
				t.Fatal("unexpected err", err)
			}
			if len(ns) > 0 {
				t.Fatal("unexpected addrs")
			}
		})
	})
}
