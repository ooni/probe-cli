package enginelocate

import (
	"github.com/ooni/probe-cli/v3/internal/geoipx"
)

type mmdbLookupper struct {
	dbPath string
}

// InitMmdbLookupper
func InitMmdbLookupper(path string) mmdbLookupper {
	return mmdbLookupper{
		dbPath: path,
	}
}

func (mmdb mmdbLookupper) LookupASN(ip string) (uint, string, error) {
	return geoipx.LookupASN(ip, mmdb.dbPath)
}

func (mmdb mmdbLookupper) LookupCC(ip string) (string, error) {
	return geoipx.LookupCC(ip, mmdb.dbPath)
}
