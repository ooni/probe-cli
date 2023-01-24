// Package geoipx contains code to use the embedded MaxMind-like databases.
package geoipx

import (
	"net"

	"github.com/ooni/probe-assets/assets"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/oschwald/maxminddb-golang"
)

// TODO(bassosimone): this would be more efficient if we'd open just
// once the database and then reuse it for every address.

// LookupASN maps [ip] to an AS number and an AS organization name.
func LookupASN(ip string) (asn uint, org string, err error) {
	asn, org = model.DefaultProbeASN, model.DefaultProbeNetworkName
	db, err := maxminddb.FromBytes(assets.OOMMDBDatabaseBytes)
	runtimex.PanicOnError(err, "cannot load embedded geoip2 database")
	defer db.Close()
	record, err := assets.OOMMDBLooup(db, net.ParseIP(ip))
	if err != nil {
		return
	}
	asn = record.AutonomousSystemNumber
	if record.AutonomousSystemOrganization != "" {
		org = record.AutonomousSystemOrganization
	}
	return
}

// LookupCC maps [ip] to a country code.
func LookupCC(ip string) (cc string, err error) {
	cc = model.DefaultProbeCC
	db, err := maxminddb.FromBytes(assets.OOMMDBDatabaseBytes)
	runtimex.PanicOnError(err, "cannot load embedded geoip2 database")
	defer db.Close()
	record, err := assets.OOMMDBLooup(db, net.ParseIP(ip))
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
