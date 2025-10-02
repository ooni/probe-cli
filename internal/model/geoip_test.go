package model

import (
	"testing"

	"github.com/oschwald/maxminddb-golang"
)

func TestGeoIPLookupperFunc(t *testing.T) {
	fx := func(reader *maxminddb.Reader, ip string) (asn uint, org string, err error) {
		return 137, "Consortium GARR", nil
	}
	lookupper := GeoIPASNLookupperFunc(fx)
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
