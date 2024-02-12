package minipipeline

import (
	"sort"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// SortDNSLookupResults sorts and returns a copy of the DNS lookup results to avoid too-much
// timing dependent churn when generating testcases for the minipipeline.
func SortDNSLookupResults(inputs []*model.ArchivalDNSLookupResult) (outputs []*model.ArchivalDNSLookupResult) {
	// copy the original slice
	outputs = append(outputs, inputs...)

	// sort using complex sorting rule
	sort.SliceStable(outputs, func(i, j int) bool {
		left, right := outputs[i], outputs[j]

		// we sort groups by resolver type to avoid the churn caused by parallel runs.
		if left.Engine < right.Engine {
			return true
		}
		if left.Engine > right.Engine {
			return false
		}

		// within the same group, we sort by ascending transaction ID
		if left.TransactionID < right.TransactionID {
			return true
		}
		if left.TransactionID > right.TransactionID {
			return false
		}

		// we want entries that are successful to appear first
		fsget := func(value *string) string {
			if value == nil {
				return ""
			}
			return *value
		}
		return fsget(left.Failure) < fsget(right.Failure)
	})

	return
}

// SortNetworkEvents is like [SortDNSLookupResults] but for network events.
func SortNetworkEvents(inputs []*model.ArchivalNetworkEvent) (outputs []*model.ArchivalNetworkEvent) {
	// copy the original slice
	outputs = append(outputs, inputs...)

	// sort using complex sorting rule
	sort.SliceStable(outputs, func(i, j int) bool {
		left, right := outputs[i], outputs[j]

		if left.Address < right.Address {
			return true
		}
		if left.Address > right.Address {
			return false
		}

		return left.TransactionID < right.TransactionID
	})

	return
}

// SortTCPConnectResults is like [SortDNSLookupResults] but for TCP connect results.
func SortTCPConnectResults(
	inputs []*model.ArchivalTCPConnectResult) (outputs []*model.ArchivalTCPConnectResult) {
	// copy the original slice
	outputs = append(outputs, inputs...)

	// sort using complex sorting rule
	sort.SliceStable(outputs, func(i, j int) bool {
		left, right := outputs[i], outputs[j]

		if left.IP < right.IP {
			return true
		}
		if left.IP > right.IP {
			return false
		}

		if left.Port < right.Port {
			return true
		}
		if left.Port > right.Port {
			return false
		}

		return left.TransactionID < right.TransactionID
	})

	return
}

// SortTLSHandshakeResults is like [SortDNSLookupResults] but for TLS handshake results.
func SortTLSHandshakeResults(
	inputs []*model.ArchivalTLSOrQUICHandshakeResult) (outputs []*model.ArchivalTLSOrQUICHandshakeResult) {
	// copy the original slice
	outputs = append(outputs, inputs...)

	// sort using complex sorting rule
	sort.SliceStable(outputs, func(i, j int) bool {
		left, right := outputs[i], outputs[j]

		if left.Address < right.Address {
			return true
		}
		if left.Address > right.Address {
			return false
		}

		return left.TransactionID < right.TransactionID
	})

	return
}
