package dslx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewDomainToResolve(t *testing.T) {
	domainToResolve := NewDomainToResolve(
		DomainName(wantDomain),
		DNSLookupOptionIDGenerator(wantIDGenerator),
		DNSLookupOptionLogger(wantLogger),
		DNSLookupOptionZeroTime(wantZeroTime),
	)
	if domainToResolve.Domain != wantDomain {
		t.Fatalf("expected: %s, got: %s", wantDomain, domainToResolve.Domain)
	}
	if domainToResolve.IDGenerator != wantIDGenerator {
		t.Fatalf("expected: %v, got: %v", wantIDGenerator, domainToResolve.IDGenerator)
	}
	if domainToResolve.Logger != wantLogger {
		t.Fatalf("expected: %v, got: %v", wantLogger, domainToResolve.Logger)
	}
	if domainToResolve.ZeroTime != wantZeroTime {
		t.Fatalf("expected: %v, got: %v", wantZeroTime, domainToResolve.ZeroTime)
	}
}

type Test struct {
	name          string
	domain        DomainToResolve
	resolver      *mocks.Resolver
	expectedErr   error
	expectedState *ResolvedAddresses
}

func TestGetaddrinfo(t *testing.T) {
	tests := []Test{
		{
			name: "GetAddrInfo w/ Invalid Domain",
			domain: DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			},
			resolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return nil, netxlite.ErrOODNSNoSuchHost
			}},
			expectedErr: netxlite.ErrOODNSNoSuchHost,
		},
		{
			name: "GetAddrInfo w/ Invalid Domain",
			domain: DomainToResolve{
				Domain:      "example.com",
				Logger:      model.DiscardLogger,
				IDGenerator: &atomic.Int64{},
				ZeroTime:    time.Time{},
			},
			resolver: &mocks.Resolver{MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return []string{"93.184.216.34"}, nil
			}},
			expectedErr: nil,
		},
	}
	getaddrinfo := DNSLookupGetaddrinfo().(*dnsLookupGetaddrinfoFunc)

	for _, test := range tests {
		getaddrinfo.resolver = test.resolver
		r := getaddrinfo.Apply(context.Background(), &test.domain)
		if r.Observations == nil || len(r.Observations) <= 0 {
			t.Fatalf("%s: expected observations, got none", test.name)
		}
		if r.Error != test.expectedErr {
			t.Fatalf("%s: expected: %s, got: %s", test.name, test.expectedErr, r.Error)
		}
		if r.Error != nil && r.State == nil {
			t.Fatalf("%s: unexpected nil-state", test.name)
		}
	}

}
