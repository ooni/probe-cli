package geolocate

import (
	"github.com/ooni/probe-cli/v3/internal/geoipx"
)

type mmdbLookupper struct{}

func (mmdbLookupper) LookupASN(ip string) (uint, string, error) {
	return geoipx.LookupASN(ip)
}

func (mmdbLookupper) LookupCC(ip string) (string, error) {
	return geoipx.LookupCC(ip)
}
