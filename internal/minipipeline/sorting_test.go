package minipipeline

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSortDNSLookupResults(t *testing.T) {
	newfailurestring := func(s string) *string {
		return &s
	}

	inputGen := func() []*model.ArchivalDNSLookupResult {
		return []*model.ArchivalDNSLookupResult{
			{
				Engine:          "udp",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "1.1.1.1:53",
				TransactionID:   5,
			},
			{
				Engine:          "udp",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "1.1.1.1:53",
				TransactionID:   5,
			},
			{
				Engine:          "udp",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "8.8.8.8:53",
				TransactionID:   3,
			},
			{
				Engine:          "udp",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "8.8.8.8:53",
				TransactionID:   3,
			},
			{
				Engine:          "doh",
				Failure:         newfailurestring("dns_no_answer"),
				QueryType:       "AAAA",
				ResolverAddress: "https://dns.google/dns-query",
				TransactionID:   2,
			},
			{
				Engine:          "doh",
				Failure:         nil,
				QueryType:       "A",
				ResolverAddress: "https://dns.google/dns-query",
				TransactionID:   2,
			},
			{
				Engine:        "getaddrinfo",
				QueryType:     "ANY",
				Failure:       nil,
				TransactionID: 1,
			},
		}
	}

	expect := []*model.ArchivalDNSLookupResult{
		{
			Engine:          "doh",
			Failure:         nil,
			QueryType:       "A",
			ResolverAddress: "https://dns.google/dns-query",
			TransactionID:   2,
		},
		{
			Engine:          "doh",
			Failure:         newfailurestring("dns_no_answer"),
			QueryType:       "AAAA",
			ResolverAddress: "https://dns.google/dns-query",
			TransactionID:   2,
		},
		{
			Engine:        "getaddrinfo",
			QueryType:     "ANY",
			Failure:       nil,
			TransactionID: 1,
		},
		{
			Engine:          "udp",
			Failure:         nil,
			QueryType:       "A",
			ResolverAddress: "8.8.8.8:53",
			TransactionID:   3,
		},
		{
			Engine:          "udp",
			Failure:         newfailurestring("dns_no_answer"),
			QueryType:       "AAAA",
			ResolverAddress: "8.8.8.8:53",
			TransactionID:   3,
		},
		{
			Engine:          "udp",
			Failure:         nil,
			QueryType:       "A",
			ResolverAddress: "1.1.1.1:53",
			TransactionID:   5,
		},
		{
			Engine:          "udp",
			Failure:         newfailurestring("dns_no_answer"),
			QueryType:       "AAAA",
			ResolverAddress: "1.1.1.1:53",
			TransactionID:   5,
		},
	}

	input := inputGen()
	output := SortDNSLookupResults(input)

	t.Run("the input should not have mutated", func(t *testing.T) {
		if diff := cmp.Diff(inputGen(), input); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("the output should be consistent with expectations", func(t *testing.T) {
		if diff := cmp.Diff(expect, output); diff != "" {
			t.Fatal(diff)
		}
	})
}
