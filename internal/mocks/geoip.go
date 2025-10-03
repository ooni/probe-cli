package mocks

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// GeoIPASNLookupper allows mocking [model.GeoIPASNLookupper].
type GeoIPASNLookupper struct {
	LookupASNFunc func(ip string) (asn uint, org string, err error)
}

var _ model.GeoIPASNLookupper = &GeoIPASNLookupper{}

// LookupASN implements model.GeoIPASNLookupper.
func (gal *GeoIPASNLookupper) LookupASN(ip string) (asn uint, org string, err error) {
	return gal.LookupASNFunc(ip)
}

// NewGeoIPASNLookupper creates a [model.GeoIPASNLookupper] from the given map.
func NewGeoIPASNLookupper(mx map[string]*model.LocationASN) model.GeoIPASNLookupper {
	return model.GeoIPASNLookupperFunc(func(ip string, dbPath string) (asn uint, org string, err error) {
		result, found := mx[ip]
		if !found {
			return 0, "", errors.New("geoip: record not found")
		}
		return result.ASNumber, result.Organization, nil
	})
}
