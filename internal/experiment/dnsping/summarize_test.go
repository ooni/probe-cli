package dnsping

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestAddressSummarizer(t *testing.T) {
	t.Run("loadQuery", func(t *testing.T) {

		// create a realistic query result with the correct addrs
		result := &model.ArchivalDNSLookupResult{
			Answers: []model.ArchivalDNSAnswer{{
				AnswerType: "A",
				IPv4:       "8.8.8.8",
			}, {
				AnswerType: "AAAA",
				IPv6:       "2001:4860:4860::8844",
			}},
			Engine:    "getaddrinfo",
			Hostname:  "dns.google",
			QueryType: "ANY",
		}

		// ingest result
		as := &addressSummarizer{}
		as.loadQuery(result, false)

		// define expectations
		expect := map[string]map[string]*summarizeAddressStats{
			"dns.google": {
				"2001:4860:4860::8844": {
					Domain:      "dns.google",
					IPAddress:   "2001:4860:4860::8844",
					ASN:         15169,
					ASOrg:       "Google LLC",
					NumResolved: 1,
				},
				"8.8.8.8": {
					Domain:      "dns.google",
					IPAddress:   "8.8.8.8",
					ASN:         15169,
					ASOrg:       "Google LLC",
					NumResolved: 1,
				},
			},
		}

		// compare
		if diff := cmp.Diff(expect, as.m); diff != "" {
			t.Fatal(diff)
		}
	})
}
