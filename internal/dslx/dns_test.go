package dslx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

/*
Test cases:
- New domain to resolve:
  - with empty domain
  - with options
*/
func TestNewDomainToResolve(t *testing.T) {
	t.Run("New domain to resolve", func(t *testing.T) {
		t.Run("with empty domain", func(t *testing.T) {
			domainToResolve := NewDomainToResolve(DomainName(""))
			if domainToResolve.Domain != "" {
				t.Fatalf("unexpected domain, want: %s, got: %s", "", domainToResolve.Domain)
			}
		})

		t.Run("with options", func(t *testing.T) {
			idGen := &atomic.Int64{}
			idGen.Add(42)
			zt := time.Now()
			domainToResolve := NewDomainToResolve(
				DomainName("www.example.com"),
				DNSLookupOptionIDGenerator(idGen),
				DNSLookupOptionLogger(model.DiscardLogger),
				DNSLookupOptionZeroTime(zt),
				DNSLookupOptionTags("antani"),
			)
			if domainToResolve.Domain != "www.example.com" {
				t.Fatalf("unexpected domain")
			}
			if domainToResolve.IDGenerator != idGen {
				t.Fatalf("unexpected id generator")
			}
			if domainToResolve.Logger != model.DiscardLogger {
				t.Fatalf("unexpected logger")
			}
			if domainToResolve.ZeroTime != zt {
				t.Fatalf("unexpected zerotime")
			}
			if diff := cmp.Diff([]string{"antani"}, domainToResolve.Tags); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

/*
Test cases:
- Get dnsLookupGetaddrinfoFunc
- Apply dnsLookupGetaddrinfoFunc
  - with nil resolver
  - with lookup error
  - with success
*/
func TestGetaddrinfo(t *testing.T) {
	t.Run("Get dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		f := DNSLookupGetaddrinfo()
		if _, ok := f.(*dnsLookupGetaddrinfoFunc); !ok {
			t.Fatal("unexpected type, want dnsLookupGetaddrinfoFunc")
		}
	})

	t.Run("Apply dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		domain := &DomainToResolve{
			Domain:      "example.com",
			Logger:      model.DiscardLogger,
			IDGenerator: &atomic.Int64{},
			Tags:        []string{"antani"},
			ZeroTime:    time.Time{},
		}

		t.Run("with nil resolver", func(t *testing.T) {
			f := dnsLookupGetaddrinfoFunc{}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately cancel the lookup
			res := f.Apply(ctx, domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error == nil {
				t.Fatal("expected an error here")
			}
		})

		t.Run("with lookup error", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			f := dnsLookupGetaddrinfoFunc{
				resolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, mockedErr
				}},
			}
			res := f.Apply(context.Background(), domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error != mockedErr {
				t.Fatalf("unexpected error type: %s", res.Error)
			}
			if res.State == nil {
				t.Fatal("unexpected nil state")
			}
			if res.State.Addresses != nil {
				t.Fatal("expected empty addresses here")
			}
		})

		t.Run("with success", func(t *testing.T) {
			f := dnsLookupGetaddrinfoFunc{
				resolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"93.184.216.34"}, nil
				}},
			}
			res := f.Apply(context.Background(), domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error != nil {
				t.Fatalf("unexpected error: %s", res.Error)
			}
			if res.State == nil {
				t.Fatal("unexpected nil state")
			}
			if len(res.State.Addresses) != 1 || res.State.Addresses[0] != "93.184.216.34" {
				t.Fatal("unexpected addresses")
			}
			if diff := cmp.Diff([]string{"antani"}, res.State.Trace.Tags()); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

/*
Test cases:
- Get dnsLookupUDPFunc
- Apply dnsLookupUDPFunc
  - with nil resolver
  - with lookup error
  - with success
*/
func TestLookupUDP(t *testing.T) {
	t.Run("Get dnsLookupUDPFunc", func(t *testing.T) {
		f := DNSLookupUDP("1.1.1.1:53")
		if _, ok := f.(*dnsLookupUDPFunc); !ok {
			t.Fatal("unexpected type, want dnsLookupUDPFunc")
		}
	})

	t.Run("Apply dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		domain := &DomainToResolve{
			Domain:      "example.com",
			Logger:      model.DiscardLogger,
			IDGenerator: &atomic.Int64{},
			Tags:        []string{"antani"},
			ZeroTime:    time.Time{},
		}

		t.Run("with nil resolver", func(t *testing.T) {
			f := dnsLookupUDPFunc{Resolver: "1.1.1.1:53"}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			res := f.Apply(ctx, domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error == nil {
				t.Fatalf("expected an error here")
			}
		})

		t.Run("with lookup error", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			f := dnsLookupUDPFunc{
				Resolver: "1.1.1.1:53",
				mockResolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, mockedErr
				}},
			}
			res := f.Apply(context.Background(), domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error != mockedErr {
				t.Fatalf("unexpected error type: %s", res.Error)
			}
			if res.State == nil {
				t.Fatal("unexpected nil state")
			}
			if res.State.Addresses != nil {
				t.Fatal("expected empty addresses here")
			}
		})

		t.Run("with success", func(t *testing.T) {
			f := dnsLookupUDPFunc{
				Resolver: "1.1.1.1:53",
				mockResolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"93.184.216.34"}, nil
				}},
			}
			res := f.Apply(context.Background(), domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error != nil {
				t.Fatalf("unexpected error: %s", res.Error)
			}
			if res.State == nil {
				t.Fatal("unexpected nil state")
			}
			if len(res.State.Addresses) != 1 || res.State.Addresses[0] != "93.184.216.34" {
				t.Fatal("unexpected addresses")
			}
			if diff := cmp.Diff([]string{"antani"}, res.State.Trace.Tags()); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}
