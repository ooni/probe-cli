package model

// GeoIPASNlookupper performs ASN lookups.
type GeoIPASNLookupper interface {
	LookupASN(ip string) (asn uint, org string, err error)
}

// GeoIPASNLookupperFunc transforms a func into a [GeoIPASNLookupper].
type GeoIPASNLookupperFunc func(ip string, dbPath string) (asn uint, org string, err error)

var _ GeoIPASNLookupper = GeoIPASNLookupperFunc(nil)

// LookupASN implements GeoIPASNLookupper.
func (fx GeoIPASNLookupperFunc) LookupASN(ip string) (asn uint, org string, err error) {
	return fx(ip, "") // TODO: this is experimental and we need to actually redesign this to take into account external geoip sources
}
