package iplookup

import (
	"github.com/ooni/probe-cli/v3/internal/geoipx"
)

type MMDBLookupper struct{}

func (MMDBLookupper) LookupASN(ip string) (uint, string, error) {
	return geoipx.LookupASN(ip)
}

func (MMDBLookupper) LookupCC(ip string) (string, error) {
	return geoipx.LookupCC(ip)
}
