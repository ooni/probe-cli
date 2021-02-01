package resolver_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

var ErrUnexpectedPunycode = errors.New("unexpected punycode value")

type CheckIDNAResolver struct {
	Addresses []string
	Error     error
	Expect    string
}

func (resolv CheckIDNAResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if resolv.Error != nil {
		return nil, resolv.Error
	}
	if hostname != resolv.Expect {
		return nil, ErrUnexpectedPunycode
	}
	return resolv.Addresses, nil
}

func (r CheckIDNAResolver) Network() string {
	return "checkidna"
}

func (r CheckIDNAResolver) Address() string {
	return ""
}

func TestIDNAResolverSuccess(t *testing.T) {
	expectedIPs := []string{"77.88.55.66"}
	resolv := resolver.IDNAResolver{Resolver: CheckIDNAResolver{
		Addresses: expectedIPs,
		Expect:    "xn--d1acpjx3f.xn--p1ai",
	}}
	addrs, err := resolv.LookupHost(context.Background(), "яндекс.рф")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expectedIPs, addrs); diff != "" {
		t.Fatal(diff)
	}
}

func TestIDNAResolverFailure(t *testing.T) {
	resolv := resolver.IDNAResolver{Resolver: CheckIDNAResolver{
		Error: errors.New("we should not arrive here"),
	}}
	// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
	addrs, err := resolv.LookupHost(context.Background(), "xn--0000h")
	if err == nil || !strings.HasPrefix(err.Error(), "idna: invalid label") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected no response here")
	}
}

func TestIDNAResolverTransportOK(t *testing.T) {
	resolv := resolver.IDNAResolver{Resolver: CheckIDNAResolver{}}
	if resolv.Network() != "idna" {
		t.Fatal("invalid network")
	}
	if resolv.Address() != "" {
		t.Fatal("invalid address")
	}
}
