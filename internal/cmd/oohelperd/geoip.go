package main

import "github.com/ooni/probe-cli/v3/internal/engine/geolocate"

//
// Contains the GeoIP task
//

// Runs the geoip task in a background goroutine
func geoipDo(mapping map[string]int, out chan<- map[string]int64) {
	geoip := make(map[string]int64)
	for addr := range mapping {
		asn, _, err := geolocate.LookupASN(addr)
		if err != nil {
			continue
		}
		geoip[addr] = int64(asn)
	}
	out <- geoip
}
