package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
)

// LocationInfo contains location information
type LocationInfo struct {
	IP          string
	ASN         uint
	NetworkName string
	CountryCode string
}

// XXX consider integration with: https://updates.maxmind.com/app/update_getfilename?product_id=GeoLite2-ASN
var geoipFiles = map[string]string{
	"GeoLite2-ASN.mmdb":     "http://geolite.maxmind.com/download/geoip/database/GeoLite2-ASN.tar.gz",
	"GeoLite2-Country.mmdb": "http://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz",
}

// Download the file to a temporary location
func downloadToTemp(url string) (string, error) {
	out, err := ioutil.TempFile(os.TempDir(), "maxmind")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary directory")
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "failed to fetch URL")
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to copy response body")
	}
	out.Close()
	resp.Body.Close()
	return out.Name(), nil
}

// DownloadGeoIPDatabaseFiles into the target directory
func DownloadGeoIPDatabaseFiles(dir string) error {
	for filename, url := range geoipFiles {
		dstPath := filepath.Join(dir, filename)

		tmpPath, err := downloadToTemp(url)
		if err != nil {
			return err
		}

		// Extract the tar.gz file
		f, err := os.Open(tmpPath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		gzf, err := gzip.NewReader(f)
		if err != nil {
			return errors.Wrap(err, "failed to create gzip reader")
		}
		tarReader := tar.NewReader(gzf)

		// Look inside of the tar for the file we need
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return errors.Wrap(err, "error extracting tar.gz")
			}
			name := header.Name
			if filepath.Base(name) == filename {
				outFile, err := os.Create(dstPath)
				if err != nil {
					return errors.Wrap(err, "error creating file")
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return errors.Wrap(err, "error reading file from tar")
				}
				outFile.Close()
				break
			}
		}
		f.Close()

	}
	return nil
}

// LookupLocation resolves an IP to a location according to the Maxmind DB
func LookupLocation(dbPath string, ipStr string) (LocationInfo, error) {
	loc := LocationInfo{IP: ipStr}

	asnDB, err := geoip2.Open(filepath.Join(dbPath, "GeoLite2-ASN.mmdb"))
	if err != nil {
		return loc, errors.Wrap(err, "failed to open ASN db")
	}
	defer asnDB.Close()

	countryDB, err := geoip2.Open(filepath.Join(dbPath, "GeoLite2-Country.mmdb"))
	if err != nil {
		return loc, errors.Wrap(err, "failed to open country db")
	}
	defer countryDB.Close()

	ip := net.ParseIP(ipStr)

	asn, err := asnDB.ASN(ip)
	if err != nil {
		return loc, err
	}
	country, err := countryDB.Country(ip)
	if err != nil {
		return loc, err
	}
	loc.ASN = asn.AutonomousSystemNumber
	loc.NetworkName = asn.AutonomousSystemOrganization
	loc.CountryCode = country.Country.IsoCode

	return loc, nil
}

type avastResponse struct {
	IP string `json:"ip"`
}

func avastLookup() (string, error) {
	var parsed = new(avastResponse)

	resp, err := http.Get("https://ip-info.ff.avast.com/v1/info")
	if err != nil {
		return "", errors.Wrap(err, "failed to perform request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}
	err = json.Unmarshal([]byte(body), &parsed)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse json")
	}

	return parsed.IP, nil
}

func akamaiLookup() (string, error) {
	// This is a domain fronted request to akamai
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://a248.e.akamai.net/", nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create NewRequest")
	}
	req.Host = "whatismyip.akamai.com"
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to perform request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "failed to read response body")
	}
	return string(body), nil
}

type lookupFunc func() (string, error)

var lookupServices = []lookupFunc{
	avastLookup,
	akamaiLookup,
}

// IPLookup gets the users IP address from a IP lookup service
func IPLookup() (string, error) {
	rand.Seed(time.Now().UnixNano())

	var (
		err   error
		ipStr string
	)

	retries := 3
	for retries > 0 {
		lookup := lookupServices[rand.Intn(len(lookupServices))]
		ipStr, err = lookup()
		if err == nil {
			return ipStr, nil
		}
		retries--
	}
	return "", errors.Wrap(err, "exceeded maximum retries")
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
