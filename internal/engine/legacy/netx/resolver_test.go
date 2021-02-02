package netx_test

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
)

func testresolverquick(t *testing.T, network, address string) {
	resolver, err := netx.NewResolver(network, address)
	if err != nil {
		t.Fatal(err)
	}
	if resolver == nil {
		t.Fatal("expected non-nil resolver here")
	}
	addrs, err := resolver.LookupHost(context.Background(), "dns.google.com")
	if err != nil {
		t.Fatalf("legacy/netx/resolver_test.go: %+v with %s/%s", err, network, address)
	}
	if addrs == nil {
		t.Fatal("expected non-nil addrs here")
	}
	var foundquad8 bool
	for _, addr := range addrs {
		// See https://github.com/ooni/probe-cli/v3/internal/engine/pull/954/checks?check_run_id=1182269025
		if addr == "8.8.8.8" || addr == "2001:4860:4860::8888" {
			foundquad8 = true
		}
	}
	if !foundquad8 {
		t.Fatalf("did not find 8.8.8.8 in ouput; output=%+v", addrs)
	}
}

func TestNewResolverUDPAddress(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8:53")
}

func TestNewResolverUDPAddressNoPort(t *testing.T) {
	testresolverquick(t, "udp", "8.8.8.8")
}

func TestNewResolverUDPDomain(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com:53")
}

func TestNewResolverUDPDomainNoPort(t *testing.T) {
	testresolverquick(t, "udp", "dns.google.com")
}

func TestNewResolverSystem(t *testing.T) {
	testresolverquick(t, "system", "")
}

func TestNewResolverTCPAddress(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8:53")
}

func TestNewResolverTCPAddressNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "8.8.8.8")
}

func TestNewResolverTCPDomain(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com:53")
}

func TestNewResolverTCPDomainNoPort(t *testing.T) {
	testresolverquick(t, "tcp", "dns.google.com")
}

func TestNewResolverDoTAddress(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("this test is not reliable in GitHub actions")
	}
	testresolverquick(t, "dot", "9.9.9.9:853")
}

func TestNewResolverDoTAddressNoPort(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("this test is not reliable in GitHub actions")
	}
	testresolverquick(t, "dot", "9.9.9.9")
}

func TestNewResolverDoTDomain(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("this test is not reliable in GitHub actions")
	}
	testresolverquick(t, "dot", "dns.quad9.net:853")
}

func TestNewResolverDoTDomainNoPort(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("this test is not reliable in GitHub actions")
	}
	testresolverquick(t, "dot", "dns.quad9.net")
}

func TestNewResolverDoH(t *testing.T) {
	testresolverquick(t, "doh", "https://cloudflare-dns.com/dns-query")
}

func TestNewResolverInvalid(t *testing.T) {
	resolver, err := netx.NewResolver(
		"antani", "https://cloudflare-dns.com/dns-query",
	)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resolver != nil {
		t.Fatal("expected a nil resolver here")
	}
}

type failingResolver struct{}

func (failingResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	return nil, io.EOF
}

func TestChainResolvers(t *testing.T) {
	fallback, err := netx.NewResolver("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	dialer := netx.NewDialer()
	resolver := netx.ChainResolvers(failingResolver{}, fallback)
	dialer.SetResolver(resolver)
	conn, err := dialer.Dial("tcp", "www.google.com:80")
	if err != nil {
		t.Fatal(err) // we don't expect error because good resolver is first
	}
	defer conn.Close()
}

func TestNewHTTPClientForDoH(t *testing.T) {
	first := netx.NewHTTPClientForDoH(
		time.Now(), handlers.NoHandler,
	)
	second := netx.NewHTTPClientForDoH(
		time.Now(), handlers.NoHandler,
	)
	if first != second {
		t.Fatal("expected to see same client here")
	}
	third := netx.NewHTTPClientForDoH(
		time.Now(), handlers.StdoutHandler,
	)
	if first == third {
		t.Fatal("expected to see different client here")
	}
}

func TestChainWrapperResolver(t *testing.T) {
	r := netx.ChainWrapperResolver{}
	if r.Address() != "" {
		t.Fatal("invalid Address")
	}
	if r.Network() != "chain" {
		t.Fatal("invalid Network")
	}
}
