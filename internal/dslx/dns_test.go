package dslx

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewDomainToResolve(t *testing.T) {
	domainToResolve := NewDomainToResolve(
		DomainName(wantDomain),
		DNSLookupOptionIDGenerator(wantIDGenerator),
		DNSLookupOptionLogger(wantLogger),
		DNSLookupOptionZeroTime(wantZeroTime),
	)
	if domainToResolve.Domain != wantDomain {
		t.Fatalf("unexpected domain, want: %s, got: %s", wantDomain, domainToResolve.Domain)
	}
	if domainToResolve.IDGenerator != wantIDGenerator {
		t.Fatalf("unexpected id generator, want: %v, got: %v", wantIDGenerator, domainToResolve.IDGenerator)
	}
	if domainToResolve.Logger != wantLogger {
		t.Fatalf("unexpected logger, want: %v, got: %v", wantLogger, domainToResolve.Logger)
	}
	if domainToResolve.ZeroTime != wantZeroTime {
		t.Fatalf("unexpected zerotime, want: %v, got: %v", wantZeroTime, domainToResolve.ZeroTime)
	}
}

func TestGetaddrinfo(t *testing.T) {
	t.Run("get dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		f := DNSLookupGetaddrinfo()
		if _, ok := f.(*dnsLookupGetaddrinfoFunc); !ok {
			t.Fatal("unexpected type, want dnsLookupGetaddrinfoFunc")
		}
	})
	t.Run("apply dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		t.Run("with lookup error", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
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
		t.Run("with nil resolver", func(t *testing.T) {
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
			f := dnsLookupGetaddrinfoFunc{}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			res := f.Apply(ctx, domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error == nil || res.Error.Error() != "interrupted" {
				t.Fatalf("expected context canceled error, got: %s", res.Error)
			}
		})
		t.Run("with success", func(t *testing.T) {
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
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
		})
	})
}

func TestLookupUDP(t *testing.T) {
	t.Run("get dnsLookupUDPFunc", func(t *testing.T) {
		f := DNSLookupUDP("1.1.1.1:53")
		if _, ok := f.(*dnsLookupUDPFunc); !ok {
			t.Fatal("unexpected type, want dnsLookupUDPFunc")
		}
	})
	t.Run("apply dnsLookupGetaddrinfoFunc", func(t *testing.T) {
		t.Run("with lookup error", func(t *testing.T) {
			mockedErr := errors.New("mocked")
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
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
		t.Run("with nil mock resolver", func(t *testing.T) {
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
			f := dnsLookupUDPFunc{Resolver: "1.1.1.1:53"}

			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			res := f.Apply(ctx, domain)
			if res.Observations == nil || len(res.Observations) <= 0 {
				t.Fatal("unexpected empty observations")
			}
			if res.Error == nil || res.Error.Error() != "interrupted" {
				t.Fatalf("expected context canceled error, got: %s", res.Error)
			}
		})
		t.Run("with success", func(t *testing.T) {
			domain := &DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			}
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
		})
	})
}
