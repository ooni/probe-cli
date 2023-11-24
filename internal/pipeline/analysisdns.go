package pipeline

import (
	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// ComputeDNSExperimentFailure computes DNSExperimentFailure.
func (ax *Analysis) ComputeDNSExperimentFailure(db *DB) {
	for _, entry := range db.DNSByTxID {
		// skip queries not for the original hostname
		if db.URLHostname != entry.QueryHostname {
			continue
		}

		// skip queries not using getaddrinfo
		if dnsNormalizeEngineName(entry.Engine) != "getaddrinfo" {
			continue
		}

		// skip successful cases
		if entry.Failure == "" {
			continue
		}

		// assign the first failure and return
		ax.DNSExperimentFailure = optional.Some(entry.Failure)
		return
	}
}

// ComputeDNSUnexpectedBogon computes DNSUnexpectedBogon.
func (ax *Analysis) ComputeDNSBogon(db *DB) {
	for _, entry := range db.WebByTxID {
		// skip all the entries without a bogon
		if !entry.IPAddressIsBogon {
			continue
		}

		// skip cases where the TH also resolved the same bogon (e.g., as of 2023-11-23, the
		// polito.it domain legitimately resolves to 192.168.59.6 and 192.168.40.1)
		if entry.DNSLookupTHXref {
			continue
		}

		// register the transaction containing the bogon.
		ax.DNSUnexpectedBogon = append(ax.DNSUnexpectedBogon, entry.TransactionID)
	}
}

// ComputeDNSUnexpectedFailure computes DNSUnexpectedFailure.
func (ax *Analysis) ComputeDNSUnexpectedFailure(db *DB) {
	// we cannot run this algorithm if the control failed or returned no IP addresses.
	if db.THDNSFailure != "" {
		return
	}

	// we cannot run this algorithm if the control returned no IP addresses.
	if len(db.THDNSAddrs) <= 0 {
		return
	}

	// inspect DNS lookup results
	for _, entry := range db.DNSByTxID {
		// skip cases without failures
		if entry.Failure == "" {
			continue
		}

		// skip cases that query the wrong domain name
		if entry.QueryHostname != db.URLHostname {
			continue
		}

		// A DoH failure is not information about the DNS blocking of the URL hostname
		// we're measuring but rather about the DoH service being blocked.
		//
		// See https://github.com/ooni/probe/issues/2274
		if entry.Engine == "doh" {
			continue
		}

		// skip cases where there's no IPv6 addresses for a domain
		if entry.QueryType == "AAAA" && entry.Failure == netxlite.FailureDNSNoAnswer {
			continue
		}

		// register the transaction as containing an unexpected DNS failure
		ax.DNSUnexpectedFailure = append(ax.DNSUnexpectedFailure, entry.TransactionID)
	}
}

func (ax *Analysis) dnsDiffHelper(db *DB, fx func(db *DB, entry *DNSObservation)) {
	// we cannot run this algorithm if the control failed or returned no IP addresses.
	if db.THDNSFailure != "" {
		return
	}

	// we cannot run this algorithm if the control returned no IP addresses.
	if len(db.THDNSAddrs) <= 0 {
		return
	}

	// inspect DNS lookup results
	for _, entry := range db.DNSByTxID {
		// skip cases witht failures
		if entry.Failure != "" {
			continue
		}

		// skip cases that query the wrong domain name
		if entry.QueryHostname != db.URLHostname {
			continue
		}

		// Note: we include DoH-resolved addresses in this comparison
		// because they should be ~as good as the TH addresses.

		// invoke user defined function
		fx(db, entry)
	}
}

// ComputeDNSUnexpectedAddr computes DNSUnexpectedAddr.
func (ax *Analysis) ComputeDNSUnexpectedAddr(db *DB) {
	ax.dnsDiffHelper(db, func(db *DB, entry *DNSObservation) {
		state := make(map[string]Origin)

		for _, addr := range entry.IPAddrs {
			state[addr] |= OriginProbe
		}

		for addr := range db.THDNSAddrs {
			state[addr] |= OriginTH
		}

		for _, flags := range state {
			if (flags & OriginTH) == 0 {
				ax.DNSUnexpectedAddr = append(ax.DNSUnexpectedAddr, entry.TransactionID)
				return
			}
		}
	})
}

// ComputeDNSUnexpectedAddrASN computes DNSUnexpectedAddrASN.
func (ax *Analysis) ComputeDNSUnexpectedAddrASN(db *DB) {
	ax.dnsDiffHelper(db, func(db *DB, entry *DNSObservation) {
		state := make(map[int64]Origin)

		for _, addr := range entry.IPAddrs {
			if asn, _, err := geoipx.LookupASN(addr); err == nil {
				state[int64(asn)] |= OriginProbe
			}
		}

		for addr := range db.THDNSAddrs {
			if asn, _, err := geoipx.LookupASN(addr); err == nil {
				state[int64(asn)] |= OriginTH
			}
		}

		for _, flags := range state {
			if (flags & OriginTH) == 0 {
				ax.DNSUnexpectedAddrASN = append(ax.DNSUnexpectedAddrASN, entry.TransactionID)
				return
			}
		}
	})
}

// ComputeDNSWithTLSHandshakeFailureAddr computes DNSWithTLSHandshakeFailureAddr.
func (ax *Analysis) ComputeDNSWithTLSHandshakeFailureAddr(db *DB) {
	ax.dnsDiffHelper(db, func(db *DB, dns *DNSObservation) {
		// walk through each resolved address in this DNS lookup
		for _, addr := range dns.IPAddrs {

			// find the corresponding endpoint measurement
			for _, epnt := range db.WebByTxID {

				// skip entries related to a different address
				if epnt.IPAddress != addr {
					continue
				}

				// skip entries where we did not attempt a TLS handshake
				if epnt.TLSHandshakeFailure.IsNone() {
					continue
				}

				// skip entries where the handshake succeded
				if epnt.TLSHandshakeFailure.Unwrap() == "" {
					continue
				}

				// find the related TH measurement
				thEpnt, good := db.THEpntByEpnt[epnt.Endpoint]

				// skip cases where there's no TH entry
				if !good {
					continue
				}

				// skip cases where the TH did not perform an handshake (a bug?)
				if thEpnt.TLSHandshakeFailure.IsNone() {
					continue
				}

				// skip cases where the TH's handshake also failed
				if thEpnt.TLSHandshakeFailure.Unwrap() != "" {
					continue
				}

				// mark the DNS transaction as bad and stop
				ax.DNSWithTLSHandshakeFailureAddr = append(ax.DNSWithTLSHandshakeFailureAddr, dns.TransactionID)
				return
			}
		}
	})
}
