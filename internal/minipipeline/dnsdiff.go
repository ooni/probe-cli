package minipipeline

import "github.com/ooni/probe-cli/v3/internal/model"

// DNSDiffFindCommonIPAddressIntersection returns the set of IP addresses that
// belong to both the measurement and the control sets.
func DNSDiffFindCommonIPAddressIntersection(measurement, control Set[string]) Set[string] {
	const (
		inMeasurement = 1 << 0
		inControl     = 1 << 1
		inBoth        = inMeasurement | inControl
	)

	ipmap := make(map[string]int)
	for _, ipAddr := range measurement.Keys() {
		ipmap[ipAddr] |= inMeasurement
	}
	for _, ipAddr := range control.Keys() {
		ipmap[ipAddr] |= inControl
	}

	state := NewSet[string]()
	for key, value := range ipmap {
		// just in case an empty string slipped through
		if key != "" && (value&inBoth) == inBoth {
			state.Add(key)
		}
	}

	return state
}

// DNSDiffFindCommonIPAddressIntersection returns the set of ASNs that belong to both the set of ASNs
// obtained from the measurement and the one obtained from the control.
func DNSDiffFindCommonASNsIntersection(
	lookupper model.GeoIPASNLookupper, measurement, control Set[string]) Set[int64] {
	const (
		inMeasurement = 1 << 0
		inControl     = 1 << 1
		inBoth        = inMeasurement | inControl
	)

	asnmap := make(map[int64]int)
	for _, ipAddr := range measurement.Keys() {
		if asn, _, err := lookupper.LookupASN(ipAddr); err == nil && asn > 0 {
			asnmap[int64(asn)] |= inMeasurement
		}
	}
	for _, ipAddr := range control.Keys() {
		if asn, _, err := lookupper.LookupASN(ipAddr); err == nil && asn > 0 {
			asnmap[int64(asn)] |= inControl
		}
	}

	state := NewSet[int64]()
	for key, value := range asnmap {
		// zero means that ASN lookup failed
		if key != 0 && (value&inBoth) == inBoth {
			state.Add(key)
		}
	}

	return state
}
