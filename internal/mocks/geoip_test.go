package mocks

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestGeoIPASNLookupper(t *testing.T) {
	lookupper := &GeoIPASNLookupper{
		LookupASNFunc: func(ip string) (uint, string, error) {
			return 137, "Consortium GARR", nil
		},
	}
	asn, org, err := lookupper.LookupASN("130.192.91.211")
	if err != nil {
		t.Fatal(err)
	}
	if asn != 137 {
		t.Fatal("invalid asn", asn)
	}
	if org != "Consortium GARR" {
		t.Fatal("invalid org", org)
	}
}

func TestNewGeoIPASNLookupper(t *testing.T) {
	lookupper := NewGeoIPASNLookupper(map[string]*model.LocationASN{
		"130.192.91.211": {
			ASNumber:     137,
			Organization: "Consortium GARR",
		},
	})

	t.Run("on success", func(t *testing.T) {
		asn, org, err := lookupper.LookupASN("130.192.91.211")
		if err != nil {
			t.Fatal(err)
		}
		if asn != 137 {
			t.Fatal("invalid asn", asn)
		}
		if org != "Consortium GARR" {
			t.Fatal("invalid org", org)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		asn, org, err := lookupper.LookupASN("130.192.91.232")
		if err == nil {
			t.Fatal("expected an error here", err)
		}
		if asn != 0 {
			t.Fatal("invalid asn", asn)
		}
		if org != "" {
			t.Fatal("invalid org", org)
		}
	})
}
