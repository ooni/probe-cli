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
	// TODO(bassosimone): now that there's a "classic" tag it would probably
	// be simpler to just always use the "classic" tag to extract.
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

		// Determine whether to keep entry depending on the IP addr origin
		switch entry.IPAddressOrigin.UnwrapOr("") {

		// If the address origin is the TH, then it does not belong to classic analysis
		case IPAddressOriginTH:
			continue

		// If the address origin is the DNS, then it depends on whether it was
		// resolved via getaddrinfo or via another resolver
		case IPAddressOriginDNS:
			if output.knownIPAddresses[ipAddr] == nil {
				continue
			}

		// If the address origin is unknown, then we assume the probe
		// already knows it, e.g., via the URL or via a subsequent redirect
		// and thus we keep this specific entry
		default:
			// nothing
		}

		// Discard all the entries where we're not fetching body
		if !entry.TagFetchBody.UnwrapOr(false) {
			continue
		}

		output.KnownTCPEndpoints[txid] = entry
	}

	// ControlFinalResponseExpectations
	output.ControlExpectations = input.ControlExpectations

	return
}
