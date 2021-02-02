package geolocate

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

type mmdbLookupper struct{}

func (mmdbLookupper) LookupASN(path, ip string) (asn uint, org string, err error) {
	asn, org = DefaultProbeASN, DefaultProbeNetworkName
	db, err := geoip2.Open(path)
	if err != nil {
		return
	}
	defer db.Close()
	record, err := db.ASN(net.ParseIP(ip))
	if err != nil {
		return
	}
	asn = record.AutonomousSystemNumber
	if record.AutonomousSystemOrganization != "" {
		org = record.AutonomousSystemOrganization
	}
	return
}

// LookupASN returns the ASN and the organization associated with the
// given ip using the ASN database at path.
func LookupASN(path, ip string) (asn uint, org string, err error) {
	return (mmdbLookupper{}).LookupASN(path, ip)
}

func (mmdbLookupper) LookupCC(path, ip string) (cc string, err error) {
	cc = DefaultProbeCC
	db, err := geoip2.Open(path)
	if err != nil {
		return
	}
	defer db.Close()
	record, err := db.Country(net.ParseIP(ip))
	if err != nil {
		return
	}
	// With MaxMind DB we used record.RegisteredCountry.IsoCode but that does
	// not seem to work with the db-ip.com database. The record is empty, at
	// least for my own IP address in Italy. --Simone (2020-02-25)
	if record.Country.IsoCode != "" {
		cc = record.Country.IsoCode
	}
	return
}
