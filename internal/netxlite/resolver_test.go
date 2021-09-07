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
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestResolverSystemNetworkAddress(t *testing.T) {
	r := &resolverSystem{}
	if r.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestResolverSystemWorksAsIntended(t *testing.T) {
	r := &resolverSystem{}
	defer r.CloseIdleConnections()
	addrs, err := r.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if addrs == nil {
		t.Fatal("expected non-nil result here")
	}
}

func TestResolverSystemDefaultTimeout(t *testing.T) {
	r := &resolverSystem{}
	if r.timeout() != 15*time.Second {
		t.Fatal("unexpected default timeout")
	}
}

func TestResolverSystemWithTimeoutAndSuccess(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	r := &resolverSystem{
		testableTimeout: 1 * time.Microsecond,
		testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
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
	wg.Wait()
}

func TestResolverSystemWithTimeoutAndFailure(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	r := &resolverSystem{
		testableTimeout: 1 * time.Microsecond,
		testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
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
	wg.Wait()
}

func TestResolverSystemWithNXDOMAIN(t *testing.T) {
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
}

func TestResolverLoggerWithSuccess(t *testing.T) {
	expected := []string{"1.1.1.1"}
	r := resolverLogger{
		Logger: log.Log,
		Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return expected, nil
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
}

func TestResolverLoggerWithFailure(t *testing.T) {
	expected := errors.New("mocked error")
	r := resolverLogger{
		Logger: log.Log,
		Resolver: &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return nil, expected
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
}

func TestResolverIDNAWorksAsIntended(t *testing.T) {
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
}

func TestResolverIDNAWithInvalidPunycode(t *testing.T) {
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
}

func TestNewResolverTypeChain(t *testing.T) {
	r := NewResolverSystem(log.Log)
	ridna, ok := r.(*resolverIDNA)
	if !ok {
		t.Fatal("invalid resolver")
	}
	rl, ok := ridna.Resolver.(*resolverLogger)
	if !ok {
		t.Fatal("invalid resolver")
	}
	if rl.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	scia, ok := rl.Resolver.(*resolverShortCircuitIPAddr)
	if !ok {
		t.Fatal("invalid resolver")
	}
	ew, ok := scia.Resolver.(*resolverErrWrapper)
	if !ok {
		t.Fatal("invalid resolver")
	}
	if _, ok := ew.Resolver.(*resolverSystem); !ok {
		t.Fatal("invalid resolver")
	}
}

func TestResolverShortCircuitIPAddrWithIPAddr(t *testing.T) {
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
}

func TestResolverShortCircuitIPAddrWithDomain(t *testing.T) {
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
}

func TestNullResolverWorksAsIntended(t *testing.T) {
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
	r.CloseIdleConnections() // should not crash
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
			if err == nil || err.Error() != errorsx.FailureEOFError {
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
}
