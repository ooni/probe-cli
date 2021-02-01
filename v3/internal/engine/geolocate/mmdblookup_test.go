package geolocate

import (
	"context"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/resources"
)

const (
	asnDBPath     = "../testdata/asn.mmdb"
	countryDBPath = "../testdata/country.mmdb"
	ipAddr        = "35.204.49.125"
)

func maybeFetchResources(t *testing.T) {
	c := &resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    "../testdata/",
	}
	if err := c.Ensure(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLookupASN(t *testing.T) {
	maybeFetchResources(t)
	asn, org, err := LookupASN(asnDBPath, ipAddr)
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

func TestLookupASNInvalidFile(t *testing.T) {
	maybeFetchResources(t)
	asn, org, err := LookupASN("/nonexistent", ipAddr)
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

func TestLookupASNInvalidIP(t *testing.T) {
	maybeFetchResources(t)
	asn, org, err := LookupASN(asnDBPath, "xxx")
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
	maybeFetchResources(t)
	cc, err := (mmdbLookupper{}).LookupCC(countryDBPath, ipAddr)
	if err != nil {
		t.Fatal(err)
	}
	if len(cc) != 2 {
		t.Fatal("does not seem a country code")
	}
}

func TestLookupCCInvalidFile(t *testing.T) {
	maybeFetchResources(t)
	cc, err := (mmdbLookupper{}).LookupCC("/nonexistent", ipAddr)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if cc != DefaultProbeCC {
		t.Fatal("expected an empty cc")
	}
}

func TestLookupCCInvalidIP(t *testing.T) {
	maybeFetchResources(t)
	cc, err := (mmdbLookupper{}).LookupCC(asnDBPath, "xxx")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if cc != DefaultProbeCC {
		t.Fatal("expected an empty cc")
	}
}
