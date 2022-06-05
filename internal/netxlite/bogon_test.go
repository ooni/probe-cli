package netxlite

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestBogonResolver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("with failure", func(t *testing.T) {
			expected := errors.New("mocked")
			reso := &BogonResolver{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return nil, expected
					},
				},
			}
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "dns.google")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if len(addrs) > 0 {
				t.Fatal("expected no addrs")
			}
		})

		t.Run("with success and no bogon", func(t *testing.T) {
			expected := []string{"8.8.8.8", "149.112.112.112"}
			reso := &BogonResolver{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return expected, nil
					},
				},
			}
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "dns.google")
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expected, addrs); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with success and bogon", func(t *testing.T) {
			reso := &BogonResolver{
				Resolver: &mocks.Resolver{
					MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
						return []string{"8.8.8.8", "10.34.34.35", "149.112.112.112"}, nil
					},
				},
			}
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "dns.google")
			if !errors.Is(err, ErrDNSBogon) {
				t.Fatal("unexpected err", err)
			}
			var ew *ErrWrapper
			if !errors.As(err, &ew) {
				t.Fatal("error has not been wrapped")
			}
			if len(addrs) > 0 {
				t.Fatal("expected no addrs")
			}
		})
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		ctx := context.Background()
		reso := &BogonResolver{}
		https, err := reso.LookupHTTPS(ctx, "dns.google")
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("unexpected err", err)
		}
		if https != nil {
			t.Fatal("expected nil https here")
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		ctx := context.Background()
		reso := &BogonResolver{}
		ns, err := reso.LookupNS(ctx, "dns.google")
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("unexpected err", err)
		}
		if len(ns) > 0 {
			t.Fatal("expected empty ns here")
		}
	})

	t.Run("Network", func(t *testing.T) {
		expected := "antani"
		reso := &BogonResolver{
			Resolver: &mocks.Resolver{
				MockNetwork: func() string {
					return expected
				},
			},
		}
		if reso.Network() != expected {
			t.Fatal("unexpected network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		expected := "antani"
		reso := &BogonResolver{
			Resolver: &mocks.Resolver{
				MockAddress: func() string {
					return expected
				},
			},
		}
		if reso.Address() != expected {
			t.Fatal("unexpected address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		reso := &BogonResolver{
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

func TestIsBogon(t *testing.T) {
	if IsBogon("antani") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("127.0.0.1") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("1.1.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("8.8.4.4") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("2001:4860:4860::8844") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("10.0.1.1") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("::1") != true {
		t.Fatal("unexpected result")
	}
}

func TestIsLoopback(t *testing.T) {
	if IsLoopback("antani") != true {
		t.Fatal("unexpected result")
	}
	if IsLoopback("127.0.0.1") != true {
		t.Fatal("unexpected result")
	}
	if IsLoopback("1.1.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("8.8.4.4") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("2001:4860:4860::8844") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("10.0.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("::1") != true {
		t.Fatal("unexpected result")
	}
}
