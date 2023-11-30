package minipipeline

// ClassicFilter takes in input a [*WebObservationsContainer] and returns in output
// another [*WebObservationsContainer] where we only keep:
//
// 1. DNS lookups using getaddrinfo;
//
// 2. IP addresses discovered using getaddrinfo;
//
// 3. endpoints using such IP addresses.
//
// We use this filter to produce a backward compatible Web Connectivity analysis
// when the input [*WebObservationsContainer] was built using LTE.
//
// The result should approximate what v0.4 would have measured.
func ClassicFilter(input *WebObservationsContainer) (output *WebObservationsContainer) {
	output = &WebObservationsContainer{
		DNSLookupFailures:  []*WebObservation{},
		DNSLookupSuccesses: []*WebObservation{},
		KnownTCPEndpoints:  map[int64]*WebObservation{},
		knownIPAddresses:   map[string]*WebObservation{},
	}

	// DNSLookupFailures
	for _, entry := range input.DNSLookupFailures {
		if !utilsEngineIsGetaddrinfo(entry.DNSEngine) {
			continue
		}
		output.DNSLookupFailures = append(output.DNSLookupFailures, entry)
	}

	// DNSLookupSuccesses & knownIPAddresses
	for _, entry := range input.DNSLookupSuccesses {
		if !utilsEngineIsGetaddrinfo(entry.DNSEngine) {
			continue
		}
		ipAddr := entry.IPAddress.Unwrap() // it MUST be there
		output.DNSLookupSuccesses = append(output.DNSLookupSuccesses, entry)
		output.knownIPAddresses[ipAddr] = entry
	}

	// KnownTCPEndpoints
	for _, entry := range input.KnownTCPEndpoints {
		ipAddr := entry.IPAddress.Unwrap() // it MUST be there
		txid := entry.EndpointTransactionID.Unwrap()
		if output.knownIPAddresses[ipAddr] == nil {
			continue
		}
		if !entry.TagFetchBody.UnwrapOr(false) {
			continue
		}
		output.KnownTCPEndpoints[txid] = entry
	}

	return
}
