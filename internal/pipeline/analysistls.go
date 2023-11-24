package pipeline

// ComputeTLSUnexpectedFailure computes TLSUnexpectedFailure.
func (ax *Analysis) ComputeTLSUnexpectedFailure(db *DB) {
	for _, entry := range db.WebByTxID {
		// skip all the entries where we did not set a TLS failure
		if entry.TLSHandshakeFailure.IsNone() {
			continue
		}

		// skip all the entries where connect succeded
		// TODO(bassosimone): say that the probe succeeds and the TH fails, then what?
		if entry.TLSHandshakeFailure.Unwrap() == "" {
			continue
		}

		// get the corresponding TH measurement
		th, good := db.THEpntByEpnt[entry.Endpoint]

		// skip if there's no TH data
		if !good {
			continue
		}

		// skip if we don't have a failure defined for TH (bug?)
		if th.TLSHandshakeFailure.IsNone() {
			continue
		}

		// skip if also the TH failed to handshake
		if th.TLSHandshakeFailure.Unwrap() != "" {
			continue
		}

		// mark this entry as problematic
		ax.TLSUnexpectedFailure = append(ax.TLSUnexpectedFailure, entry.TransactionID)
	}
}
