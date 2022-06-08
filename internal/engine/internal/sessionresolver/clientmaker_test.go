package sessionresolver

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type fakeDNSClientMaker struct {
	reso        model.Resolver
	err         error
	savedConfig netx.Config
	savedURL    string
}

func (c *fakeDNSClientMaker) Make(config netx.Config, URL string) (model.Resolver, error) {
	c.savedConfig = config
	c.savedURL = URL
	return c.reso, c.err
}

func TestClientMakerWithOverride(t *testing.T) {
	m := &fakeDNSClientMaker{err: io.EOF}
	reso := &Resolver{dnsClientMaker: m}
	out, err := reso.clientmaker().Make(netx.Config{}, "https://dns.google/dns-query")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestClientDefaultWithCancelledContext(t *testing.T) {
	reso := &Resolver{}
	re, err := reso.clientmaker().Make(netx.Config{}, "https://dns.google/dns-query")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := re.LookupHost(ctx, "dns.google")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil output")
	}
}
