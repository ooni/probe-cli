package netxlite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverGetaddrinfo(t *testing.T) {
	t.Run("RequiresPadding", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		if txp.RequiresPadding() {
			t.Fatal("expected false")
		}
	})

	t.Run("Network", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		if txp.Network() != getaddrinfoResolverNetwork() {
			t.Fatal("unexpected Network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		if txp.Address() != "" {
			t.Fatal("unexpected Address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		txp.CloseIdleConnections() // does not crash
	})

	t.Run("check default timeout", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		if txp.timeout() != 15*time.Second {
			t.Fatal("unexpected default timeout")
		}
	})

	t.Run("check default lookup host func not nil", func(t *testing.T) {
		txp := &dnsOverGetaddrinfoTransport{}
		if txp.lookupfn() == nil {
			t.Fatal("expected non-nil func here")
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("with invalid query type", func(t *testing.T) {
			txp := &dnsOverGetaddrinfoTransport{
				testableLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return []string{"8.8.8.8"}, "dns.google", nil
				},
			}
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google", dns.TypeA, false)
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if !errors.Is(err, ErrNoDNSTransport) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp")
			}
		})

		t.Run("with success", func(t *testing.T) {
			txp := &dnsOverGetaddrinfoTransport{
				testableLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return []string{"8.8.8.8"}, "dns.google", nil
				},
			}
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google", dns.TypeANY, false)
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := resp.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("invalid addrs")
			}
			if resp.Query() != query {
				t.Fatal("invalid query")
			}
			if len(resp.Bytes()) != 0 {
				t.Fatal("invalid response bytes")
			}
			if resp.Rcode() != 0 {
				t.Fatal("invalid rcode")
			}
			https, err := resp.DecodeHTTPS()
			if !errors.Is(err, ErrNoDNSTransport) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("expected nil https")
			}
			ns, err := resp.DecodeNS()
			if !errors.Is(err, ErrNoDNSTransport) {
				t.Fatal("unexpected err", err)
			}
			if len(ns) != 0 {
				t.Fatal("expected zero-length ns")
			}
		})

		t.Run("with timeout and success", func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			done := make(chan interface{})
			txp := &dnsOverGetaddrinfoTransport{
				testableTimeout: 1 * time.Microsecond,
				testableLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					defer wg.Done()
					<-done
					return []string{"8.8.8.8"}, "dns.google", nil
				},
			}
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google", dns.TypeANY, false)
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("invalid resp")
			}
			close(done)
			wg.Wait()
		})

		t.Run("with timeout and failure", func(t *testing.T) {
			wg := &sync.WaitGroup{}
			wg.Add(1)
			done := make(chan interface{})
			txp := &dnsOverGetaddrinfoTransport{
				testableTimeout: 1 * time.Microsecond,
				testableLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					defer wg.Done()
					<-done
					return nil, "", errors.New("no such host")
				},
			}
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google", dns.TypeANY, false)
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("not the error we expected", err)
			}
			if resp != nil {
				t.Fatal("invalid resp")
			}
			close(done)
			wg.Wait()
		})

		t.Run("with NXDOMAIN", func(t *testing.T) {
			txp := &dnsOverGetaddrinfoTransport{
				testableLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return nil, "", ErrOODNSNoSuchHost
				},
			}
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google", dns.TypeANY, false)
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
				t.Fatal("not the error we expected", err)
			}
			if resp != nil {
				t.Fatal("invalid resp")
			}
		})
	})
}

func TestDNSOverGetaddrinfoResponse(t *testing.T) {
	t.Run("Query works as intended", func(t *testing.T) {
		t.Run("when query is not nil", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{},
				cname: "",
				query: &mocks.DNSQuery{},
			}
			out := resp.Query()
			if out != resp.query {
				t.Fatal("unexpected query")
			}
		})

		t.Run("when query is nil", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{},
				cname: "",
				query: nil, // oops
			}
			panicked := false
			func() {
				defer func() {
					if recover() != nil {
						panicked = true
					}
				}()
				_ = resp.Query()
			}()
			if !panicked {
				t.Fatal("did not panic")
			}
		})
	})

	t.Run("Bytes works as intended", func(t *testing.T) {
		resp := &dnsOverGetaddrinfoResponse{
			addrs: []string{},
			cname: "",
			query: nil,
		}
		if len(resp.Bytes()) > 0 {
			t.Fatal("unexpected bytes")
		}
	})

	t.Run("Rcode works as intended", func(t *testing.T) {
		resp := &dnsOverGetaddrinfoResponse{
			addrs: []string{},
			cname: "",
			query: nil,
		}
		if resp.Rcode() != 0 {
			t.Fatal("unexpected rcode")
		}
	})

	t.Run("DecodeHTTPS works as intended", func(t *testing.T) {
		resp := &dnsOverGetaddrinfoResponse{
			addrs: []string{},
			cname: "",
			query: nil,
		}
		out, err := resp.DecodeHTTPS()
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("unexpected err")
		}
		if out != nil {
			t.Fatal("unexpected result")
		}
	})

	t.Run("DecodeLookupHost works as intended", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{
					"1.1.1.1", "1.0.0.1",
				},
				cname: "",
				query: nil,
			}
			out, err := resp.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(resp.addrs, out); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("on failure", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{},
				cname: "",
				query: nil,
			}
			out, err := resp.DecodeLookupHost()
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("unexpected err")
			}
			if len(out) > 0 {
				t.Fatal("unexpected addrs")
			}
		})
	})

	t.Run("DecodeNS works as intended", func(t *testing.T) {
		resp := &dnsOverGetaddrinfoResponse{
			addrs: []string{},
			cname: "",
			query: nil,
		}
		out, err := resp.DecodeNS()
		if !errors.Is(err, ErrNoDNSTransport) {
			t.Fatal("unexpected err")
		}
		if len(out) != 0 {
			t.Fatal("unexpected result")
		}
	})

	t.Run("DecodeCNAME works as intended", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{},
				cname: "antani",
				query: nil,
			}
			out, err := resp.DecodeCNAME()
			if err != nil {
				t.Fatal(err)
			}
			if out != resp.cname {
				t.Fatal("unexpected cname")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			resp := &dnsOverGetaddrinfoResponse{
				addrs: []string{},
				cname: "",
				query: nil,
			}
			out, err := resp.DecodeCNAME()
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("unexpected err")
			}
			if out != "" {
				t.Fatal("unexpected cname")
			}
		})
	})
}
