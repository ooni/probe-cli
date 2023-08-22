package model

// GeoIPASNlookupper performs ASN lookups.
type GeoIPASNLookupper interface {
	LookupASN(ip string) (asn uint, org string, err error)
}

// GeoIPASNLookupperFunc transforms a func into a [GeoIPASNLookupper].
type GeoIPASNLookupperFunc func(ip string) (asn uint, org string, err error)

var _ GeoIPASNLookupper = GeoIPASNLookupperFunc(nil)

// LookupASN implements GeoIPASNLookupper.
func (fx GeoIPASNLookupperFunc) LookupASN(ip string) (asn uint, org string, err error) {
	return fx(ip)
}
