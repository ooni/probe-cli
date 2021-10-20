package geolocate

import "testing"

const ipAddr = "8.8.8.8"

func TestLookupASN(t *testing.T) {
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
}

func TestLookupASNInvalidIP(t *testing.T) {
	asn, org, err := LookupASN("xxx")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if asn != DefaultProbeASN {
		t.Fatal("expected a zero ASN")
	}
	if org != DefaultProbeNetworkName {
		t.Fatal("expected an empty org")
	}
}

func TestLookupCC(t *testing.T) {
	cc, err := (mmdbLookupper{}).LookupCC(ipAddr)
	if err != nil {
		t.Fatal(err)
	}
	if cc != "US" {
		t.Fatal("invalid country code", cc)
	}
}

func TestLookupCCInvalidIP(t *testing.T) {
	cc, err := (mmdbLookupper{}).LookupCC("xxx")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if cc != DefaultProbeCC {
		t.Fatal("expected an empty cc")
	}
}
