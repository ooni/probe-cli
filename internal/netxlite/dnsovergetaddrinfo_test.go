package netxlite

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
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
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8"}, nil
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
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"8.8.8.8"}, nil
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
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					defer wg.Done()
					<-done
					return []string{"8.8.8.8"}, nil
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
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					defer wg.Done()
					<-done
					return nil, errors.New("no such host")
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
				testableLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, ErrOODNSNoSuchHost
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
