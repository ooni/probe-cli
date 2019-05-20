package utils

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/version"
	"github.com/ooni/probe-engine/geoiplookup/iplookup"
	"github.com/ooni/probe-engine/geoiplookup/mmdblookup"
	"github.com/ooni/probe-engine/httpx/httpx"
	"github.com/ooni/probe-engine/resources"
)

// LocationInfo contains location information
type LocationInfo struct {
	IP          string
	ASN         uint
	NetworkName string
	CountryCode string
}

// MaybeDownloadGeoIPDatabaseFiles into the target directory. This function
// is idempotent and won't donwload the files if they're already in place
// and their SHA256 matches the expected one.
func MaybeDownloadGeoIPDatabaseFiles(dir string) error {
	return (&resources.Client{
		HTTPClient: httpx.NewTracingProxyingClient(
			log.Log, http.ProxyFromEnvironment,
		),
		Logger: log.Log,
		UserAgent: version.UserAgent,
		WorkDir: dir,
	}).Ensure(context.Background())
}

// LookupLocation resolves an IP to a location according to the Maxmind DB
func LookupLocation(dbPath string, ipStr string) (info LocationInfo, err error) {
	info.IP = ipStr
	info.ASN, info.NetworkName, err = mmdblookup.LookupASN(
		filepath.Join(dbPath, resources.ASNDatabaseName), ipStr,
	)
	if err != nil {
		return
	}
	info.CountryCode, err = mmdblookup.LookupCC(
		filepath.Join(dbPath, resources.CountryDatabaseName), ipStr,
	)
	return
}

// IPLookup gets the users IP address from a IP lookup service
func IPLookup() (string, error) {
	return (&iplookup.Client{
		HTTPClient: httpx.NewTracingProxyingClient(
			log.Log, nil, // no proxy, we need to connect directly
		),
		Logger: log.Log,
		UserAgent: version.UserAgent,
	}).Do(context.Background())
}

// GeoIPLookup does a geoip lookup and returns location information
func GeoIPLookup(dbPath string) (*LocationInfo, error) {
	ipStr, err := IPLookup()
	if err != nil {
		return nil, err
	}
	location, err := LookupLocation(dbPath, ipStr)
	if err != nil {
		return nil, err
	}
	return &location, nil
}
