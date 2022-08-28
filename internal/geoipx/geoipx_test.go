package geoipx

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const ipAddr = "8.8.8.8"

func TestLookupASN(t *testing.T) {
	t.Run("with valid IP address", func(t *testing.T) {
		asn, org, err := LookupASN(ipAddr)
		if err != nil {
			t.Fatal(err)
		}
		if asn != 15169 {
			t.Fatal("unexpected ASN value", asn)
		}
		if org != "Google LLC" {
			t.Fatal("unexpected org value", org)
		}
	})

	t.Run("with invalid IP address", func(t *testing.T) {
		asn, org, err := LookupASN("xxx")
		if err == nil {
			t.Fatal("expected an error here")
		}
		if asn != model.DefaultProbeASN {
			t.Fatal("expected a zero ASN")
		}
		if org != model.DefaultProbeNetworkName {
			t.Fatal("expected an empty org")
		}
	})
}

func TestLookupCC(t *testing.T) {
	t.Run("with valid IP address", func(t *testing.T) {
		cc, err := LookupCC(ipAddr)
		if err != nil {
			t.Fatal(err)
		}
		if cc != "US" {
			t.Fatal("invalid country code", cc)
		}
	})

	t.Run("with invalid IP address", func(t *testing.T) {
		cc, err := LookupCC("xxx")
		if err == nil {
			t.Fatal("expected an error here")
		}
		if cc != model.DefaultProbeCC {
			t.Fatal("expected an empty cc")
		}
	})
}
