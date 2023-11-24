package pipeline

import "github.com/ooni/probe-cli/v3/internal/netxlite"

// ComputeTCPUnexpectedFailure computes TCPUnexpectedFailure.
func (ax *Analysis) ComputeTCPUnexpectedFailure(db *DB) {
	for _, entry := range db.webByTxID {
		// skip all the entries where we did not set a TCP failure
		if entry.TCPConnectFailure.IsNone() {
			continue
		}

		// skip all the entries where connect succeded
		// TODO(bassosimone): say that the probe succeeds and the TH fails, then what?
		if entry.TCPConnectFailure.Unwrap() == nil {
			continue
		}

		// get the corresponding TH measurement
		th, good := db.thEpntByEpnt[entry.Endpoint]

		// skip if there's no TH data
		if !good {
			continue
		}

		// skip if we don't have a failure defined for TH (bug?)
		if th.TCPConnectFailure.IsNone() {
			continue
		}

		// skip if also the TH failed to connect
		if th.TCPConnectFailure.Unwrap() != nil {
			continue
		}

		// ignore failures that are most likely caused by a broken IPv6 network stack
		if ipv6, err := netxlite.IsIPv6(entry.IPAddress); err == nil && ipv6 {
			failure := th.TCPConnectFailure.Unwrap()
			likelyFalsePositive := (*failure == netxlite.FailureNetworkUnreachable ||
				*failure == netxlite.FailureHostUnreachable)
			if likelyFalsePositive {
				continue
			}
		}

		// mark this entry as problematic
		ax.TCPUnexpectedFailure = append(ax.TCPUnexpectedFailure, entry.TransactionID)
	}
}
