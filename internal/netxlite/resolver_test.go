package netxlite

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
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
	_ = errWrapper.Resolver.(*resolverSystem)
}

func TestNewResolverUDP(t *testing.T) {
	d := NewDialerWithoutResolver(log.Log)
	resolver := NewResolverUDP(log.Log, d, "1.1.1.1:53")
	idna := resolver.(*resolverIDNA)
	logger := idna.Resolver.(*resolverLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	shortCircuit := logger.Resolver.(*resolverShortCircuitIPAddr)
	errWrapper := shortCircuit.Resolver.(*resolverErrWrapper)
	serio := errWrapper.Resolver.(*SerialResolver)
	txp := serio.Transport().(*DNSOverUDP)
	if txp.Address() != "1.1.1.1:53" {
		t.Fatal("invalid address")
	}
}

func TestResolverSystem(t *testing.T) {
	t.Run("Network and Address", func(t *testing.T) {
		r := &resolverSystem{}
		if r.Network() != "system" {
			t.Fatal("invalid Network")
		}
		if r.Address() != "" {
			t.Fatal("invalid Address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		r := &resolverSystem{}
		r.CloseIdleConnections() // to cover it
	})

	t.Run("check default timeout", func(t *testing.T) {
		r := &resolverSystem{}
		if r.timeout() != 15*time.Second {
			t.Fatal("unexpected default timeout")
		}
	})

	t.Run("check default lookup host func not nil", func(t *testing.T) {
		r := &resolverSystem{}
		if r.lookupHost() == nil {
			t.Fatal("expected non-nil func here")
		}
	})

	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with success", func(t *testing.T) {
			r := &resolverSystem{
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8"}, nil
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "example.antani")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("invalid addrs")
			}
		})

		t.Run("with timeout and success", func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			done := make(chan interface{})
			r := &resolverSystem{
				testableTimeout: 1 * time.Microsecond,
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					defer wg.Done()
					<-done
					return []string{"8.8.8.8"}, nil
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "example.antani")
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("invalid addrs")
			}
			close(done)
			wg.Wait()
		})

		t.Run("with timeout and failure", func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			done := make(chan interface{})
			r := &resolverSystem{
				testableTimeout: 1 * time.Microsecond,
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					defer wg.Done()
					<-done
					return nil, errors.New("no such host")
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "example.antani")
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("invalid addrs")
			}
			close(done)
			wg.Wait()
		})

		t.Run("with NXDOMAIN", func(t *testing.T) {
			r := &resolverSystem{
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, errors.New("no such host")
				},
			}
			ctx := context.Background()
			addrs, err := r.LookupHost(ctx, "example.antani")
			if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
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
}

func TestNullResolver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		r := &nullResolver{}
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
		r := &nullResolver{}
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
}
