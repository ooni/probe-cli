package main

//
// Contains the GeoIP task
//

import (
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
)

// Runs the geoip task in a background goroutine
func geoipDo(wg *sync.WaitGroup, mapping map[string]int, out chan<- map[string]int64) {
	defer wg.Done()
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
