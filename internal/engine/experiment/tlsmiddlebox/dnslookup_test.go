package tlsmiddlebox

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSLookup_success(t *testing.T) {
	resolver := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return []string{"1.1.1.1"}, nil
		},
		MockNetwork: func() string {
			return "doh"
		},
		MockAddress: func() string {
			return "https://dns.google/dns-query"
		},
	}
	m := NewExperimentMeasurer(Config{})
	ctx := context.Background()
	out, addrs, err := m.DNSLookup(ctx, "www.example.com", resolver)
	expected := model.ArchivalDNSLookupResult{
		Answers: []model.ArchivalDNSAnswer{
			{
				AnswerType: "A",
				IPv4:       "1.1.1.1",
			},
		},
		Failure:         nil,
		Engine:          "doh",
		Hostname:        "www.example.com",
		ResolverAddress: "https://dns.google/dns-query",
	}
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if len(addrs) != 1 {
		t.Fatal("expected 1 address")
	}
	if diff := cmp.Diff(*out, expected); diff != "" {
		t.Fatal(diff)
	}
}

func TestDNSLookup_failure(t *testing.T) {
	t.Run("with cancelled context", func(t *testing.T) {
		m := NewExperimentMeasurer(Config{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		out, addrs, err := m.DNSLookup(ctx, "www.example.com", nil)
		expectedFailure := netxlite.FailureInterrupted
		expected := model.ArchivalDNSLookupResult{
			Answers:         []model.ArchivalDNSAnswer{},
			Failure:         &expectedFailure,
			Engine:          "system",
			Hostname:        "www.example.com",
			ResolverAddress: "",
		}
		if err == nil || err.Error() != netxlite.FailureInterrupted {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 0 {
			t.Fatal("did not expect addresses")
		}
		if diff := cmp.Diff(*out, expected); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestDNSLookup_resolver(t *testing.T) {
	t.Run("with default resolver", func(t *testing.T) {
		ctx := context.Background()
		m := NewExperimentMeasurer(Config{
			ResolverURL: "",
		})
		out, addrs, err := m.DNSLookup(ctx, "1.1.1.1", nil)
		expected := model.ArchivalDNSLookupResult{
			Answers: []model.ArchivalDNSAnswer{
				{
					AnswerType: "A",
					IPv4:       "1.1.1.1",
				},
			},
			Failure:         nil,
			Engine:          "doh",
			Hostname:        "1.1.1.1",
			ResolverAddress: "https://mozilla.cloudflare-dns.com/dns-query",
		}
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 1 {
			t.Fatal("expected 1 address, got", len(addrs))
		}
		if diff := cmp.Diff(*out, expected); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("with custom resolver", func(t *testing.T) {
		ctx := context.Background()
		m := NewExperimentMeasurer(Config{
			ResolverURL: "https://dns.google/dns-query",
		})
		out, addrs, err := m.DNSLookup(ctx, "1.1.1.1", nil)
		expected := model.ArchivalDNSLookupResult{
			Answers: []model.ArchivalDNSAnswer{
				{
					AnswerType: "A",
					IPv4:       "1.1.1.1",
				},
			},
			Failure:         nil,
			Engine:          "doh",
			Hostname:        "1.1.1.1",
			ResolverAddress: "https://dns.google/dns-query",
		}
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 1 {
			t.Fatal("expected 1 address, got", len(addrs))
		}
		if diff := cmp.Diff(*out, expected); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("with system resolver", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		m := NewExperimentMeasurer(Config{
			ResolverURL: "",
		})
		out, addrs, err := m.DNSLookup(ctx, "example.com", nil)
		expectedFailure := netxlite.FailureInterrupted
		expected := model.ArchivalDNSLookupResult{
			Answers:         []model.ArchivalDNSAnswer{},
			Failure:         &expectedFailure,
			Engine:          "system",
			Hostname:        "example.com",
			ResolverAddress: "",
		}
		if err == nil || err.Error() != expectedFailure {
			t.Fatal("unexpected error", err)
		}
		if len(addrs) != 0 {
			t.Fatal("expected 0 addresses, got", len(addrs))
		}
		if diff := cmp.Diff(*out, expected); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestWriteToArchival(t *testing.T) {
	resolver := mocks.Resolver{
		MockNetwork: func() string {
			return "doh"
		},
		MockAddress: func() string {
			return "https://dns.google/dns-query"
		},
	}
	addrs := []string{"1.1.1.1", "2001:4860:4860::8844"}
	expected := model.ArchivalDNSLookupResult{
		Answers: []model.ArchivalDNSAnswer{
			{
				AnswerType: "A",
				IPv4:       "1.1.1.1",
			},
			{
				AnswerType: "AAAA",
				IPv6:       "2001:4860:4860::8844",
			},
		},
		Failure:         nil,
		Engine:          "doh",
		Hostname:        "www.example.com",
		ResolverAddress: "https://dns.google/dns-query",
	}
	out := WriteDNSToArchival(&resolver, "www.example.com", addrs, nil)
	if diff := cmp.Diff(*out, expected); diff != "" {
		t.Fatal(diff)
	}
}

func TestAnswersFromAddrs(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []model.ArchivalDNSAnswer
	}{{
		name:  "with valid input",
		input: []string{"1.1.1.1", "2001:4860:4860::8844"},
		want: []model.ArchivalDNSAnswer{
			{
				AnswerType: "A",
				IPv4:       "1.1.1.1",
			},
			{
				AnswerType: "AAAA",
				IPv6:       "2001:4860:4860::8844",
			},
		},
	}, {
		name:  "with invalid input",
		input: []string{"1.1.1.1.1", "2001:4860:4860::8844"},
		want: []model.ArchivalDNSAnswer{
			{
				AnswerType: "AAAA",
				IPv6:       "2001:4860:4860::8844",
			},
		},
	}, {
		name:  "with empty input",
		input: []string{},
		want:  []model.ArchivalDNSAnswer{},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := answersFromAddrs(tt.input)
			if diff := cmp.Diff(out, tt.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
