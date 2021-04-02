package geolocate

import "testing"

const ipAddr = "35.204.49.125"

func TestLookupASN(t *testing.T) {
	asn, org, err := LookupASN(ipAddr)
	if err != nil {
		t.Fatal(err)
	}
	if asn <= 0 {
		t.Fatal("unexpected ASN value")
	}
	if len(org) <= 0 {
		t.Fatal("unexpected org value")
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
	if len(cc) != 2 {
		t.Fatal("does not seem a country code")
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
