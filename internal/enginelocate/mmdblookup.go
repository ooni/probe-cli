package enginelocate

import (
	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/oschwald/maxminddb-golang"
)

type mmdbLookupper struct {
	reader *maxminddb.Reader
}

// InitMmdbLookupper
func InitMmdbLookupper(path string) (mmdbLookupper, error) {
	if path == "" {
		return mmdbLookupper{}, nil

	}
	db, err := maxminddb.Open(path)
	if err != nil {
		return mmdbLookupper{}, err
	}

	return mmdbLookupper{
		reader: db,
	}, nil
}

func (mmdb mmdbLookupper) LookupASN(ip string) (uint, string, error) {
	return geoipx.LookupASN(mmdb.reader, ip)
}

func (mmdb mmdbLookupper) LookupCC(ip string) (string, error) {
	return geoipx.LookupCC(mmdb.reader, ip)
}
