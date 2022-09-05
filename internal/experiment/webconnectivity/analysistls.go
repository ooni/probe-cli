package webconnectivity

//
// TLS analysis
//

import "github.com/ooni/probe-cli/v3/internal/model"

// analysisTLSToplevel is the toplevel analysis function for TLS.
//
// This algorithm aims to flag the TLS endpoints that failed unreasonably
// compared to what the TH has observed for the same endpoints.
func (tk *TestKeys) analysisTLSToplevel(logger model.Logger) {
	// if we don't have a control result, do nothing.
	if tk.Control == nil || len(tk.Control.TLSHandshake) <= 0 {
		return
	}

	// walk the list of probe results and compare with TH results
	for _, entry := range tk.TLSHandshakes {
		// skip successful entries
		failure := entry.Failure
		if failure == nil {
			continue // did not fail
		}
		epnt := entry.Address

		// TODO(bassosimone,kelmenhorst): if, in the future, we choose to
		// adapt this code to QUIC, we need to remember to treat EHOSTUNREACH
		// and ENETUNREACH specially when the IP address is IPv6.

		// obtain the corresponding endpoint
		ctrl, found := tk.Control.TLSHandshake[epnt]
		if !found {
			continue // only the probe tested this, so hard to say anything...
		}
		if ctrl.Failure != nil {
			// If the TH failed as well, don't set XBlockingFlags. Performing
			// precise error mapping should be a job for the pipeline.
			continue
		}
		logger.Warnf(
			"TLS: endpoint %s is blocked (see #%d): %s",
			epnt,
			entry.TransactionID,
			*failure,
		)
		tk.BlockingFlags |= analysisFlagTLSBlocking
	}
}
